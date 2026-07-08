import React from "react";
import { AlertTriangle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { ResponseView, UsageInfo } from "@/lib/conversation";
import { MessageCard } from "./MessageCard";
import { SectionLabel } from "./RequestOverview";

/**
 * Semantic view of a normalized response: model, stop reason, token usage,
 * the assistant's output blocks, and (for streams) an event summary.
 */
export function ResponseOverview({ view }: { view: ResponseView }) {
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        {view.model && <Badge variant="secondary">{view.model}</Badge>}
        {view.stopReason && (
          <Badge variant="outline" className="font-mono font-normal">
            <span>
              <span className="text-muted-foreground">stop=</span>
              {view.stopReason}
            </span>
          </Badge>
        )}
        {view.usage && <UsageChips usage={view.usage} />}
      </div>

      {view.error && (
        <div className="flex items-start gap-2 rounded-lg border border-destructive-foreground/25 bg-destructive p-3">
          <AlertTriangle className="h-4 w-4 text-destructive-foreground shrink-0 mt-0.5" />
          <p className="text-sm text-destructive-foreground whitespace-pre-wrap break-words">
            {view.error}
          </p>
        </div>
      )}

      {view.message && view.message.blocks.length > 0 && (
        <div>
          <SectionLabel>Output</SectionLabel>
          <MessageCard message={view.message} />
        </div>
      )}

      {view.eventCounts && Object.keys(view.eventCounts).length > 0 && (
        <div>
          <SectionLabel>Stream events</SectionLabel>
          <div className="flex flex-wrap gap-1.5">
            {Object.entries(view.eventCounts).map(([type, count]) => (
              <Badge
                key={type}
                variant="outline"
                className="font-mono font-normal"
              >
                {type}
                <span className="text-muted-foreground">×{count}</span>
              </Badge>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

const USAGE_LABELS: [keyof UsageInfo, string][] = [
  ["input", "in"],
  ["output", "out"],
  ["cacheRead", "cache read"],
  ["cacheWrite", "cache write"],
  ["reasoning", "reasoning"],
  ["total", "total"],
];

function UsageChips({ usage }: { usage: UsageInfo }) {
  return (
    <>
      {USAGE_LABELS.map(([key, label]) => {
        const value = usage[key];
        if (value == null) return null;
        return (
          <Badge key={key} variant="outline" className="font-mono font-normal">
            <span className="text-muted-foreground">{label}</span>
            {value.toLocaleString()}
          </Badge>
        );
      })}
    </>
  );
}
