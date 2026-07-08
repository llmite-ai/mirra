import React from "react";
import { format } from "date-fns";
import { Recording } from "@/lib/api";
import {
  getStatusColor,
  getProviderStyles,
  getProviderLabel,
} from "@/lib/styles";

interface RecordingMetadataProps {
  recording: Recording;
}

/**
 * Displays key metadata: timestamp, provider, model, duration, status
 */
export function RecordingMetadata({ recording }: RecordingMetadataProps) {
  const body: any = recording.request.body;
  const model = body && typeof body === "object" ? body.model : undefined;

  return (
    <div className="flex flex-wrap gap-x-10 gap-y-3 mb-6">
      <MetadataField label="Timestamp">
        {format(new Date(recording.timestamp), "MMM d, yyyy HH:mm:ss")}
      </MetadataField>
      <MetadataField label="Provider">
        <span
          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getProviderStyles(recording.provider)}`}
        >
          {getProviderLabel(recording.provider)}
        </span>
      </MetadataField>
      {model && (
        <MetadataField label="Model">
          <span className="font-mono">{model}</span>
        </MetadataField>
      )}
      <MetadataField label="Status">
        <span
          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(recording.response.status)}`}
        >
          {recording.response.status}
        </span>
      </MetadataField>
      <MetadataField label="Duration">
        {recording.timing.duration_ms}ms
      </MetadataField>
      {recording.response.streaming && (
        <MetadataField label="Transport">
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground ring-1 ring-inset ring-border">
            SSE stream
          </span>
        </MetadataField>
      )}
    </div>
  );
}

function MetadataField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <label className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
        {label}
      </label>
      <p className="text-sm mt-1.5">{children}</p>
    </div>
  );
}
