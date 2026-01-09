import React, { useEffect } from "react";
import { List, FolderTree } from "lucide-react";

type ViewMode = "list" | "sessions";

interface ViewModeToggleProps {
  value: ViewMode;
  onChange: (value: ViewMode) => void;
}

export default function ViewModeToggle({ value, onChange }: ViewModeToggleProps) {
  // Persist preference to localStorage
  useEffect(() => {
    localStorage.setItem("mirra-view-mode", value);
  }, [value]);

  // Load preference from localStorage on mount
  useEffect(() => {
    const saved = localStorage.getItem("mirra-view-mode");
    if (saved === "list" || saved === "sessions") {
      onChange(saved);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="inline-flex items-center rounded-md bg-muted p-1">
      <button
        onClick={() => onChange("list")}
        className={`
          inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded transition-colors
          ${
            value === "list"
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }
        `}
      >
        <List className="h-4 w-4" />
        Recordings
      </button>
      <button
        onClick={() => onChange("sessions")}
        className={`
          inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded transition-colors
          ${
            value === "sessions"
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          }
        `}
      >
        <FolderTree className="h-4 w-4" />
        Sessions
      </button>
    </div>
  );
}
