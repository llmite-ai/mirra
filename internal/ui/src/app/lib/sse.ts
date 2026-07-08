/**
 * Client-side parsing of recorded SSE stream bodies.
 *
 * Recordings store streaming responses as the raw accumulated SSE text.
 * Parsing happens here (not on the server) so the semantic views can work
 * off the full event list for any provider, including ones the Go parser
 * doesn't know about.
 */

export interface SSEEvent {
  /** Value of the `event:` line, if present */
  event?: string;
  /** Concatenated `data:` payload */
  data: string;
  /** Parsed JSON payload, when the data is valid JSON */
  json?: any;
}

/**
 * Returns true if a string body looks like an SSE stream.
 */
export function looksLikeSSE(body: string): boolean {
  const head = body.slice(0, 2000).trimStart();
  return /^(event|data):/m.test(head);
}

/**
 * Returns true if a string body looks like binary/mangled content
 * (e.g. compressed bytes recorded before decompression existed).
 */
export function looksBinary(body: string): boolean {
  const sample = body.slice(0, 512);
  if (sample.includes("�")) return true;
  let control = 0;
  for (let i = 0; i < sample.length; i++) {
    const c = sample.charCodeAt(i);
    if (c < 32 && c !== 9 && c !== 10 && c !== 13) control++;
  }
  return control > sample.length * 0.05;
}

/**
 * Parses raw SSE text into a list of events. Tolerates \r\n line endings,
 * multi-line data fields, and trailing partial events.
 */
export function parseSSE(text: string): SSEEvent[] {
  const events: SSEEvent[] = [];
  let eventName: string | undefined;
  let dataLines: string[] = [];

  const flush = () => {
    if (eventName === undefined && dataLines.length === 0) return;
    const data = dataLines.join("\n");
    const event: SSEEvent = { event: eventName, data };
    if (data && data !== "[DONE]") {
      try {
        event.json = JSON.parse(data);
      } catch {
        // Non-JSON data stays raw
      }
    }
    events.push(event);
    eventName = undefined;
    dataLines = [];
  };

  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine;
    if (line === "") {
      flush();
      continue;
    }
    if (line.startsWith(":")) continue; // comment
    const colon = line.indexOf(":");
    const field = colon === -1 ? line : line.slice(0, colon);
    let value = colon === -1 ? "" : line.slice(colon + 1);
    if (value.startsWith(" ")) value = value.slice(1);

    if (field === "event") {
      eventName = value;
    } else if (field === "data") {
      dataLines.push(value);
    }
    // id/retry fields are irrelevant for recordings
  }
  flush();

  return events;
}

/**
 * Counts events by type (the `event:` name, falling back to the JSON
 * payload's `type` field).
 */
export function countEvents(events: SSEEvent[]): Record<string, number> {
  const counts: Record<string, number> = {};
  for (const e of events) {
    const key =
      e.event || e.json?.type || (e.data === "[DONE]" ? "[DONE]" : "unknown");
    counts[key] = (counts[key] || 0) + 1;
  }
  return counts;
}
