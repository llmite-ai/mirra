import React from "react";
import { ChevronDown, ChevronRight, AlertCircle } from "lucide-react";
import { format } from "date-fns";
import { SessionGroup } from "@/lib/api";
import { getProviderStyles, getErrorBadgeStyles } from "@/lib/styles";
import { formatTraceId, formatTimeRange } from "@/lib/formatters";

interface SessionGroupRowProps {
  group: SessionGroup;
  isExpanded: boolean;
  onToggle: () => void;
}

export default function SessionGroupRow({
  group,
  isExpanded,
  onToggle,
}: SessionGroupRowProps) {
  return (
    <button
      onClick={onToggle}
      className={`
        w-full text-left p-4 border-2 rounded-md transition-colors
        bg-muted/20 border-muted-foreground/20
        hover:bg-muted/30 hover:border-muted-foreground/30
        focus:outline-none focus:ring-2 focus:ring-primary/50
      `}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 flex-1">
          {/* Expand/Collapse Icon */}
          {isExpanded ? (
            <ChevronDown className="h-5 w-5 text-muted-foreground flex-shrink-0" />
          ) : (
            <ChevronRight className="h-5 w-5 text-muted-foreground flex-shrink-0" />
          )}

          {/* Session Info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-3 mb-1">
              <span className="font-mono text-sm font-medium text-foreground">
                Session: {formatTraceId(group.trace_id || group.session_id, 16)}
              </span>
              {group.has_errors && (
                <span
                  className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${getErrorBadgeStyles()}`}
                >
                  <AlertCircle className="h-3 w-3" />
                  Has errors
                </span>
              )}
            </div>

            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <span>
                {group.request_count} request{group.request_count !== 1 ? "s" : ""}
              </span>
              <span>
                {format(new Date(group.first_timestamp), "MMM d")} |{" "}
                {formatTimeRange(group.first_timestamp, group.last_timestamp)}
              </span>
            </div>
          </div>

          {/* Provider Badges */}
          <div className="flex items-center gap-2 flex-shrink-0">
            {group.providers.map((provider) => (
              <span
                key={provider}
                className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${getProviderStyles(provider)}`}
              >
                {provider}
              </span>
            ))}
          </div>
        </div>
      </div>
    </button>
  );
}
