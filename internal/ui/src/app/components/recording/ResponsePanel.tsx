import React from "react";
import { Recording } from "@/lib/api";
import { BodyView } from "./BodyView";
import { HeadersSection } from "./HeadersSection";

interface ResponsePanelProps {
  recording: Recording;
}

/**
 * Displays response details: headers and body. Status lives in the header,
 * and copying is handled by the tab bar.
 */
export function ResponsePanel({ recording }: ResponsePanelProps) {
  return (
    <div className="bg-card border rounded-lg shadow-elev overflow-hidden p-4 space-y-3">
      <HeadersSection headers={recording.response.headers} />

      <div>
        <label className="text-sm font-medium text-muted-foreground">
          Body
        </label>
        <div className="mt-1">
          <BodyView recording={recording} which="response" />
        </div>
      </div>
    </div>
  );
}
