import React from "react";

/**
 * The mirra mark: a solid chevron (the request) meeting its outlined
 * reflection (the recording) across a mirror axis. The reflection carries
 * the brand frost blue; everything else follows the current text color.
 */
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <polygon points="3,7 13,16 3,25" fill="currentColor" />
      <rect
        x="15.25"
        y="5"
        width="1.5"
        height="22"
        fill="currentColor"
        opacity="0.55"
      />
      <polygon
        points="29,7 19,16 29,25"
        fill="none"
        stroke="oklch(0.65 0.12 220)"
        strokeWidth="2"
      />
    </svg>
  );
}

/**
 * Mark + wordmark. The second "r" renders mirrored so the double-r pair
 * reflects around its own axis.
 */
export function Logo() {
  return (
    <span className="inline-flex items-center gap-2 text-foreground">
      <LogoMark className="h-6 w-6" />
      <span
        className="text-xl font-semibold tracking-tight leading-none select-none"
        style={{ fontFamily: '"Google Sans Code", monospace' }}
      >
        mi<span>r</span>
        <span className="inline-block" style={{ transform: "scaleX(-1)" }}>
          r
        </span>
        a
      </span>
    </span>
  );
}
