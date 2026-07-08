import React from "react";
import { format } from "date-fns";
import { formatRelativeTime } from "@/lib/formatters";

interface RelativeTimeProps {
  /** ISO timestamp to render relative to now. */
  timestamp: string;
  className?: string;
}

/** How often the displayed value is recomputed, in milliseconds. */
const TICK_MS = 15_000;

/**
 * Renders a timestamp as a terse, self-updating relative time (e.g. "5s ago").
 * Re-renders on an interval so values stay fresh, and exposes the absolute
 * time as a hover tooltip.
 */
export function RelativeTime({ timestamp, className }: RelativeTimeProps) {
  const [, setTick] = React.useState(0);

  React.useEffect(() => {
    const id = window.setInterval(() => setTick((t) => t + 1), TICK_MS);
    return () => window.clearInterval(id);
  }, []);

  const relative = formatRelativeTime(timestamp);
  if (!relative) return null;

  return (
    <span
      className={className}
      title={format(new Date(timestamp), "MMM d, yyyy HH:mm:ss")}
    >
      {relative}
    </span>
  );
}
