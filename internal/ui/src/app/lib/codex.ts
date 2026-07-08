/**
 * Normalizers for the OpenAI Responses API (/v1/responses), which is what
 * Codex speaks — both with an API key (provider "openai") and through a
 * ChatGPT subscription (provider "chatgpt").
 */

import {
  Block,
  Message,
  RequestView,
  ResponseView,
  ToolDefinition,
  UsageInfo,
  joinText,
} from "./conversation";
import { SSEEvent, countEvents } from "./sse";

const HANDLED_REQUEST_KEYS = new Set([
  "model",
  "stream",
  "instructions",
  "input",
  "tools",
]);

function parseArguments(args: unknown): unknown {
  if (typeof args !== "string") return args;
  try {
    return JSON.parse(args);
  } catch {
    return args;
  }
}

function contentPartBlock(part: any): Block {
  if (part == null) return { kind: "other", label: "empty", raw: part };
  if (typeof part === "string") return { kind: "text", text: part };
  switch (part.type) {
    case "input_text":
    case "output_text":
    case "summary_text":
    case "text":
      return { kind: "text", text: part.text ?? "" };
    case "refusal":
      return { kind: "text", text: part.refusal ?? "[refusal]" };
    case "input_image":
      return {
        kind: "image",
        src:
          typeof part.image_url === "string"
            ? part.image_url
            : part.image_url?.url,
      };
    default:
      return { kind: "other", label: part.type ?? "unknown", raw: part };
  }
}

/**
 * Maps a Responses API input/output item to a message. Function calls and
 * their outputs become their own messages so the tool loop reads in order.
 */
function itemToMessage(item: any): Message | null {
  if (item == null) return null;
  if (typeof item === "string") {
    return { role: "user", blocks: [{ kind: "text", text: item }] };
  }

  const type = item.type ?? (item.role ? "message" : undefined);
  switch (type) {
    case "message": {
      const blocks: Block[] = Array.isArray(item.content)
        ? item.content.map(contentPartBlock)
        : [{ kind: "text", text: String(item.content ?? "") } as Block];
      return { role: item.role ?? "user", blocks };
    }
    case "function_call":
      return {
        role: "assistant",
        blocks: [
          {
            kind: "tool_call",
            id: item.call_id ?? item.id,
            name: item.name ?? "function",
            input: parseArguments(item.arguments),
          },
        ],
      };
    case "custom_tool_call":
      return {
        role: "assistant",
        blocks: [
          {
            kind: "tool_call",
            id: item.call_id ?? item.id,
            name: item.name ?? "custom_tool",
            input: item.input,
          },
        ],
      };
    case "local_shell_call":
      return {
        role: "assistant",
        blocks: [
          {
            kind: "tool_call",
            id: item.call_id ?? item.id,
            name: "local_shell",
            input: item.action,
          },
        ],
      };
    case "function_call_output":
    case "custom_tool_call_output":
    case "local_shell_call_output": {
      const text =
        typeof item.output === "string" ? item.output : joinText(item.output);
      return {
        role: "tool",
        blocks: [
          {
            kind: "tool_result",
            id: item.call_id,
            text,
            raw: text ? undefined : item.output,
          },
        ],
      };
    }
    case "reasoning": {
      const summary = joinText(item.summary) || joinText(item.content);
      if (!summary && item.encrypted_content) return null; // opaque reasoning replay, nothing to show
      return {
        role: "assistant",
        blocks: [{ kind: "thinking", text: summary || "[reasoning]" }],
      };
    }
    case "web_search_call":
      return {
        role: "assistant",
        blocks: [
          {
            kind: "tool_call",
            id: item.id,
            name: "web_search",
            input: item.action ?? item.query,
          },
        ],
      };
    default:
      return {
        role: item.role ?? "assistant",
        blocks: [{ kind: "other", label: type ?? "unknown", raw: item }],
      };
  }
}

function responsesTool(t: any): ToolDefinition {
  if (t?.type === "function" || t?.name) {
    return {
      name: t.name ?? "function",
      description: t.description,
      schema: t.parameters,
    };
  }
  return {
    name: t?.type ?? "unknown",
    description: t?.description,
    schema: undefined,
  };
}

export function isResponsesRequestBody(body: any): boolean {
  return (
    body != null &&
    typeof body === "object" &&
    (Array.isArray(body.input) ||
      typeof body.input === "string" ||
      typeof body.instructions === "string")
  );
}

export function normalizeResponsesRequest(body: any): RequestView {
  const input = body.input;
  const items: any[] = Array.isArray(input)
    ? input
    : input != null
      ? [input]
      : [];
  const messages = items
    .map(itemToMessage)
    .filter((m): m is Message => m != null);

  const params: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(body)) {
    if (!HANDLED_REQUEST_KEYS.has(key) && value != null) {
      params[key] = value;
    }
  }

  return {
    model: body.model,
    stream: body.stream === true,
    system:
      typeof body.instructions === "string" ? body.instructions : undefined,
    tools: Array.isArray(body.tools) ? body.tools.map(responsesTool) : [],
    messages,
    params,
  };
}

function responsesUsage(usage: any): UsageInfo | undefined {
  if (usage == null || typeof usage !== "object") return undefined;
  return {
    input: usage.input_tokens,
    output: usage.output_tokens,
    total: usage.total_tokens,
    cacheRead: usage.input_tokens_details?.cached_tokens,
    reasoning: usage.output_tokens_details?.reasoning_tokens,
  };
}

/** Normalizes a full Responses API response object (non-streaming or from response.completed). */
export function normalizeResponsesResponse(body: any): ResponseView | null {
  if (body == null || typeof body !== "object") return null;
  if (body.error && !body.output) {
    return { error: body.error?.message ?? "Unknown API error" };
  }
  if (!Array.isArray(body.output)) return null;

  const blocks: Block[] = [];
  for (const item of body.output) {
    const msg = itemToMessage(item);
    if (msg) blocks.push(...msg.blocks);
  }

  let stopReason: string | undefined = body.status;
  if (body.status === "incomplete" && body.incomplete_details?.reason) {
    stopReason = `incomplete: ${body.incomplete_details.reason}`;
  }

  return {
    model: body.model,
    id: body.id,
    stopReason,
    message: { role: "assistant", blocks },
    usage: responsesUsage(body.usage),
    error: body.error?.message,
  };
}

/**
 * Rebuilds the response from a recorded Responses API SSE stream. The final
 * response.completed/failed/incomplete event carries status and usage, but
 * the chatgpt backend sends it with an empty output array, so the output
 * blocks come from response.output_item.done events (text deltas are the
 * last resort for truncated streams).
 */
export function reconstructResponsesStream(
  events: SSEEvent[],
): ResponseView | null {
  const eventCounts = countEvents(events);

  const doneItems: any[] = [];
  let deltaText = "";
  let model: string | undefined;
  let id: string | undefined;
  let finalView: ResponseView | null = null;
  for (const e of events) {
    const data = e.json;
    if (!data) continue;
    switch (data.type) {
      case "response.created":
        model = data.response?.model;
        id = data.response?.id;
        break;
      case "response.output_item.done":
        doneItems.push(data.item);
        break;
      case "response.output_text.delta":
        deltaText += data.delta ?? "";
        break;
      case "response.completed":
      case "response.failed":
      case "response.incomplete":
        finalView = normalizeResponsesResponse(data.response);
        break;
    }
  }

  if (finalView?.message?.blocks.length) {
    return { ...finalView, eventCounts };
  }

  if (!finalView && doneItems.length === 0 && !deltaText && !model) return null;

  const blocks: Block[] = [];
  for (const item of doneItems) {
    const msg = itemToMessage(item);
    if (msg) blocks.push(...msg.blocks);
  }
  if (blocks.length === 0 && deltaText) {
    blocks.push({ kind: "text", text: deltaText });
  }

  return {
    model: finalView?.model ?? model,
    id: finalView?.id ?? id,
    stopReason: finalView?.stopReason ?? "stream truncated",
    usage: finalView?.usage,
    error: finalView?.error,
    message: { role: "assistant", blocks },
    eventCounts,
  };
}
