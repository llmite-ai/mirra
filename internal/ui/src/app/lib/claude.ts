/**
 * Normalizers for the Claude Messages API (/v1/messages).
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

/** Request params surfaced as chips; everything structural is handled separately. */
const HANDLED_REQUEST_KEYS = new Set([
  "model",
  "stream",
  "system",
  "messages",
  "tools",
]);

function claudeContentBlock(part: any): Block {
  if (part == null) return { kind: "other", label: "empty", raw: part };
  if (typeof part === "string") return { kind: "text", text: part };

  switch (part.type) {
    case "text":
      return { kind: "text", text: part.text ?? "" };
    case "thinking":
      return { kind: "thinking", text: part.thinking ?? "" };
    case "redacted_thinking":
      return { kind: "thinking", text: "[redacted thinking]" };
    case "tool_use":
    case "server_tool_use":
      return {
        kind: "tool_call",
        id: part.id,
        name: part.name ?? part.type,
        input: part.input,
      };
    case "tool_result": {
      const text = joinText(part.content);
      return {
        kind: "tool_result",
        id: part.tool_use_id,
        isError: part.is_error === true,
        text,
        raw: text ? undefined : part.content,
      };
    }
    case "image": {
      const source = part.source ?? {};
      if (source.type === "base64" && source.data) {
        return {
          kind: "image",
          mediaType: source.media_type,
          src: `data:${source.media_type ?? "image/png"};base64,${source.data}`,
        };
      }
      return { kind: "image", mediaType: source.media_type, src: source.url };
    }
    default:
      return { kind: "other", label: part.type ?? "unknown", raw: part };
  }
}

function claudeMessage(msg: any): Message {
  const blocks: Block[] = Array.isArray(msg?.content)
    ? msg.content.map(claudeContentBlock)
    : [{ kind: "text", text: String(msg?.content ?? "") } as Block];
  return { role: msg?.role ?? "unknown", blocks };
}

export function isClaudeMessagesBody(body: any): boolean {
  return (
    body != null && typeof body === "object" && Array.isArray(body.messages)
  );
}

export function normalizeClaudeRequest(body: any): RequestView {
  const tools: ToolDefinition[] = Array.isArray(body.tools)
    ? body.tools.map((t: any) => ({
        name: t?.name ?? t?.type ?? "unknown",
        description: t?.description,
        schema: t?.input_schema,
      }))
    : [];

  const params: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(body)) {
    if (!HANDLED_REQUEST_KEYS.has(key) && value != null) {
      params[key] = value;
    }
  }

  return {
    model: body.model,
    stream: body.stream === true,
    system: joinText(body.system) || undefined,
    tools,
    messages: body.messages.map(claudeMessage),
    params,
  };
}

// Only defined fields are set so partial usage (message_start has input,
// message_delta has output) merges without clobbering.
function claudeUsage(usage: any): UsageInfo | undefined {
  if (usage == null || typeof usage !== "object") return undefined;
  const info: UsageInfo = {};
  if (usage.input_tokens != null) info.input = usage.input_tokens;
  if (usage.output_tokens != null) info.output = usage.output_tokens;
  if (usage.cache_read_input_tokens != null)
    info.cacheRead = usage.cache_read_input_tokens;
  if (usage.cache_creation_input_tokens != null)
    info.cacheWrite = usage.cache_creation_input_tokens;
  return Object.keys(info).length > 0 ? info : undefined;
}

/** Normalizes a non-streaming /v1/messages response body. */
export function normalizeClaudeResponse(body: any): ResponseView | null {
  if (body == null || typeof body !== "object") return null;
  if (body.type === "error") {
    return { error: body.error?.message ?? "Unknown API error" };
  }
  if (body.type !== "message" || !Array.isArray(body.content)) return null;

  return {
    model: body.model,
    id: body.id,
    stopReason: body.stop_reason ?? undefined,
    message: {
      role: body.role ?? "assistant",
      blocks: body.content.map(claudeContentBlock),
    },
    usage: claudeUsage(body.usage),
  };
}

/**
 * Rebuilds the response message from a recorded Claude SSE stream:
 * content blocks are opened by content_block_start, grown by
 * content_block_delta, and stop_reason/usage arrive in message_delta.
 */
export function reconstructClaudeStream(
  events: SSEEvent[],
): ResponseView | null {
  const view: ResponseView = { eventCounts: countEvents(events) };
  const blocks: Block[] = [];
  // Partial tool inputs accumulate as JSON text until the block closes
  const partialJSON: Record<number, string> = {};
  let sawMessage = false;

  for (const e of events) {
    const data = e.json;
    if (!data) continue;
    switch (data.type) {
      case "message_start": {
        sawMessage = true;
        view.model = data.message?.model;
        view.id = data.message?.id;
        view.usage = claudeUsage(data.message?.usage);
        break;
      }
      case "content_block_start": {
        blocks[data.index] = claudeContentBlock(data.content_block);
        break;
      }
      case "content_block_delta": {
        const block = blocks[data.index];
        const delta = data.delta;
        if (!block || !delta) break;
        if (delta.type === "text_delta" && block.kind === "text") {
          block.text += delta.text ?? "";
        } else if (
          delta.type === "thinking_delta" &&
          block.kind === "thinking"
        ) {
          block.text += delta.thinking ?? "";
        } else if (
          delta.type === "input_json_delta" &&
          block.kind === "tool_call"
        ) {
          partialJSON[data.index] =
            (partialJSON[data.index] ?? "") + (delta.partial_json ?? "");
        }
        break;
      }
      case "content_block_stop": {
        const block = blocks[data.index];
        const partial = partialJSON[data.index];
        if (block?.kind === "tool_call" && partial) {
          try {
            block.input = JSON.parse(partial);
          } catch {
            block.input = partial;
          }
        }
        break;
      }
      case "message_delta": {
        if (data.delta?.stop_reason) view.stopReason = data.delta.stop_reason;
        const usage = claudeUsage(data.usage);
        if (usage) view.usage = { ...view.usage, ...usage };
        break;
      }
      case "error": {
        view.error = data.error?.message ?? "Stream error";
        break;
      }
    }
  }

  if (!sawMessage && !view.error) return null;
  view.message = { role: "assistant", blocks: blocks.filter(Boolean) };
  return view;
}
