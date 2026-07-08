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
 * Formats a timestamp as a terse relative time from `now`.
 * Units mirror {@link formatTimespan} (s/m/h) so both read consistently.
 * @example formatRelativeTime(fiveSecondsAgo) // "5s ago"
 * @example formatRelativeTime(twoHoursAgo)    // "2h ago"
 * @returns "" when the timestamp cannot be parsed, so callers can omit it.
 */
export function formatRelativeTime(
  timestamp: string,
  now: number = Date.now(),
): string {
  const then = new Date(timestamp).getTime();
  if (!Number.isFinite(then)) return "";

  const seconds = Math.round((now - then) / 1000);
  // Future timestamps (clock skew) read cleaner as "just now" than "-3s ago".
  if (seconds < 1) return "just now";
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  const weeks = Math.floor(days / 7);
  if (weeks < 5) return `${weeks}w ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(days / 365)}y ago`;
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
