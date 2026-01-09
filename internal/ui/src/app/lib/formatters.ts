/**
 * Formatting utilities for consistent data display
 */

/**
 * Formats bytes into human-readable size
 * @example formatBytes(1536) // "1.5 KB"
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/**
 * Truncates ID to first 8 characters for display
 * @example truncateId("abcd1234-5678-90ef") // "abcd1234"
 */
export function truncateId(id: string, length: number = 8): string {
  return id.substring(0, length);
}

/**
 * Formats JSON with proper indentation
 * Falls back to string representation if JSON.stringify fails
 */
export function formatJSON(obj: any): string {
  try {
    return JSON.stringify(obj, null, 2);
  } catch {
    return String(obj);
  }
}

/**
 * Formats body content - handles both string and object bodies
 */
export function formatBody(body: any): string {
  if (typeof body === "string") {
    return body;
  }
  return formatJSON(body);
}

/**
 * Formats a time range from start to end timestamps
 * @example formatTimeRange("2024-11-30T15:23:01Z", "2024-11-30T15:45:22Z")
 *          // "Nov 30, 15:23 - 15:45 (22m)"
 */
export function formatTimeRange(start: string, end: string): string {
  const startDate = new Date(start);
  const endDate = new Date(end);

  const startTime = startDate.toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  });
  const endTime = endDate.toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  });

  const durationMs = endDate.getTime() - startDate.getTime();
  const durationMin = Math.round(durationMs / 60000);

  return `${startTime} - ${endTime} (${durationMin}m)`;
}

/**
 * Formats trace ID for display by truncating long IDs
 * @example formatTraceId("abc123def456ghi789") // "abc123de..."
 */
export function formatTraceId(id: string, maxLength: number = 12): string {
  if (id.length <= maxLength) {
    return id;
  }
  return `${id.substring(0, maxLength)}...`;
}
