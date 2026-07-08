import React from "react";
import { Recording } from "@/lib/api";
import { BodyView } from "./BodyView";
import { HeadersSection } from "./HeadersSection";

interface RequestPanelProps {
  recording: Recording;
}

/**
 * Displays request details: headers and body. The method + endpoint live in
 * the header, and copying is handled by the tab bar.
 */
export function RequestPanel({ recording }: RequestPanelProps) {
  return (
    <div className="bg-card border rounded-lg shadow-elev overflow-hidden p-4 space-y-3">
      <HeadersSection headers={recording.request.headers} />

      <div>
        <label className="text-sm font-medium text-muted-foreground">
          Body
        </label>
        <div className="mt-1">
          <BodyView recording={recording} which="request" />
        </div>
      </div>
    </div>
  );
}
