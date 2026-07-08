/**
 * Canonical conversation model.
 *
 * Provider-specific request/response bodies (Claude Messages API, OpenAI
 * Responses API) are normalized into these types so a single set of view
 * components can render all of them.
 */

export type Block =
  | { kind: "text"; text: string }
  | { kind: "thinking"; text: string }
  | { kind: "tool_call"; id?: string; name: string; input: unknown }
  | {
      kind: "tool_result";
      id?: string;
      isError?: boolean;
      text: string;
      /** Non-text result payload, when the content wasn't plain text */
      raw?: unknown;
    }
  | { kind: "image"; mediaType?: string; src?: string }
  | { kind: "other"; label: string; raw: unknown };

export interface Message {
  role: string;
  blocks: Block[];
}

export interface ToolDefinition {
  name: string;
  description?: string;
  schema?: unknown;
}

export interface RequestView {
  model?: string;
  stream?: boolean;
  /** System prompt / instructions text */
  system?: string;
  tools: ToolDefinition[];
  messages: Message[];
  /** Remaining request parameters worth surfacing (max_tokens, temperature, ...) */
  params: Record<string, unknown>;
}

export interface UsageInfo {
  input?: number;
  output?: number;
  cacheRead?: number;
  cacheWrite?: number;
  reasoning?: number;
  total?: number;
}

export interface ResponseView {
  model?: string;
  id?: string;
  stopReason?: string;
  message?: Message;
  usage?: UsageInfo;
  /** Present when the view was reconstructed from an SSE stream */
  eventCounts?: Record<string, number>;
  /** Set when the body is an API error payload */
  error?: string;
}

/** Joins content that may be a plain string or a list of text parts. */
export function joinText(content: unknown): string {
  if (typeof content === "string") return content;
  if (Array.isArray(content)) {
    return content
      .map((part) => {
        if (typeof part === "string") return part;
        if (part && typeof part.text === "string") return part.text;
        return "";
      })
      .filter(Boolean)
      .join("\n\n");
  }
  return "";
}

/** Counts blocks of a given kind across messages. */
export function countBlocks(messages: Message[], kind: Block["kind"]): number {
  let n = 0;
  for (const m of messages) {
    for (const b of m.blocks) {
      if (b.kind === kind) n++;
    }
  }
  return n;
}
