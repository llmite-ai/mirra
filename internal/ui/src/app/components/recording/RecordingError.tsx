import React from "react";

interface RecordingErrorProps {
  error: string;
}

/**
 * Displays error message in a styled alert box
 */
export function RecordingError({ error }: RecordingErrorProps) {
  return (
    <div className="p-4 rounded-lg bg-rose-500/10 border border-rose-500/25">
      <label className="text-[11px] font-semibold uppercase tracking-[0.08em] text-rose-700 dark:text-rose-300">
        Error
      </label>
      <p className="text-sm text-rose-700 dark:text-rose-300 mt-1 font-mono">
        {error}
      </p>
    </div>
  );
}
