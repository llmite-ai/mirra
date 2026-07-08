import React, { useLayoutEffect, useRef, useState } from "react";

interface ExpandableProps {
  children: React.ReactNode;
  /** Collapsed height in pixels */
  maxHeight?: number;
  className?: string;
}

/**
 * Clamps tall content (system prompts, tool results) behind a fade and a
 * show-more control. Content shorter than the clamp renders untouched.
 */
export function Expandable({
  children,
  maxHeight = 260,
  className,
}: ExpandableProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [overflows, setOverflows] = useState(false);

  useLayoutEffect(() => {
    if (ref.current) {
      setOverflows(ref.current.scrollHeight > maxHeight + 20);
    }
  }, [maxHeight, children]);

  return (
    <div className={className}>
      <div
        ref={ref}
        className="relative overflow-hidden"
        style={expanded ? undefined : { maxHeight }}
      >
        {children}
        {overflows && !expanded && (
          <div className="absolute bottom-0 left-0 right-0 h-10 bg-gradient-to-t from-card to-transparent pointer-events-none" />
        )}
      </div>
      {overflows && (
        <button
          onClick={() => setExpanded(!expanded)}
          className="mt-1 text-xs text-primary hover:underline"
        >
          {expanded ? "Show less" : "Show more"}
        </button>
      )}
    </div>
  );
}
