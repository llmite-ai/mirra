/**
 * Session grouping helpers.
 */

/** Formats the wall-clock span of a session, e.g. "4m 12s". */
export function formatTimespan(first: string, last: string): string {
  const ms = new Date(last).getTime() - new Date(first).getTime();
  if (!Number.isFinite(ms) || ms < 0) return "—";
  const seconds = Math.round(ms / 1000);
  if (seconds < 1) return "<1s";
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
}
