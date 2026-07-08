import React, { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

/**
 * Collapsible key/value listing of HTTP headers.
 */
export function HeadersSection({
  headers,
}: {
  headers: Record<string, string[]>;
}) {
  const [open, setOpen] = useState(false);
  const entries = Object.entries(headers ?? {}).sort(([a], [b]) =>
    a.localeCompare(b),
  );

  return (
    <div>
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
      >
        {open ? (
          <ChevronDown className="h-4 w-4" />
        ) : (
          <ChevronRight className="h-4 w-4" />
        )}
        Headers <span className="text-xs">({entries.length})</span>
      </button>
      {open && (
        <div className="mt-1 border bg-muted/20 p-3 overflow-x-auto">
          <table className="text-xs font-mono">
            <tbody>
              {entries.map(([key, values]) => (
                <tr key={key} className="align-top">
                  <td className="pr-4 py-0.5 text-chart-2 whitespace-nowrap">
                    {key}
                  </td>
                  <td className="py-0.5 break-all">{values.join(", ")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
