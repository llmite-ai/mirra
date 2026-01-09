import React, { useMemo } from "react";
import { Loader2 } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router";
import { fetchRecording, fetchParsedRecording, fetchSessionGroup } from "@/lib/api";
import { RecordingHeader } from "./RecordingHeader";
import { RecordingMetadata } from "./RecordingMetadata";
import { RecordingError } from "./RecordingError";
import { RecordingTabs } from "./RecordingTabs";
import { RequestPanel } from "./RequestPanel";
import { ResponsePanel } from "./ResponsePanel";
import { ParsedResponsePanel } from "./ParsedResponsePanel";

interface RecordingDetailProps {
  recordingId: string;
}

const TABS = [
  { id: "request", label: "Request" },
  { id: "response", label: "Response" },
  { id: "parsed", label: "Parsed Response" },
];

/**
 * Main orchestrator for recording detail view
 * Handles data fetching and coordinates child components
 */
export default function RecordingDetail({ recordingId }: RecordingDetailProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = searchParams.get("tab") || "request";

  const setActiveTab = (tab: string) => {
    setSearchParams(
      (prev) => {
        const newParams = new URLSearchParams(prev);
        newParams.set("tab", tab);
        return newParams;
      },
      { replace: true }
    );
  };

  const { data: recording, isLoading: isLoadingRecording } = useQuery({
    queryKey: ["recording", recordingId],
    queryFn: () => fetchRecording(recordingId),
    enabled: !!recordingId,
  });

  // Extract trace ID from sentry-trace header
  const traceId = useMemo(() => {
    if (!recording?.request?.headers) return null;

    // Look for sentry-trace header (case-insensitive)
    const sentryTrace = Object.entries(recording.request.headers).find(
      ([key]) => key.toLowerCase() === "sentry-trace"
    );

    if (!sentryTrace || !sentryTrace[1] || sentryTrace[1].length === 0) {
      return null;
    }

    // Extract trace ID (before first dash)
    const traceValue = sentryTrace[1][0];
    return traceValue.split("-")[0];
  }, [recording]);

  // Fetch session group if we have a trace ID
  const { data: sessionData } = useQuery({
    queryKey: ["session-for-recording", traceId],
    queryFn: () => fetchSessionGroup(traceId!),
    enabled: !!traceId,
  });

  // Build session context if we have session data
  const sessionContext = useMemo(() => {
    if (!sessionData || !recordingId) return undefined;

    const position = sessionData.group.recording_ids.indexOf(recordingId) + 1;
    return {
      traceId: sessionData.group.trace_id || sessionData.group.session_id,
      position,
      total: sessionData.group.request_count,
    };
  }, [sessionData, recordingId]);

  const {
    data: parsedData,
    isLoading: isParsing,
    error: parseError,
  } = useQuery({
    queryKey: ["parsed", recordingId],
    queryFn: () => fetchParsedRecording(recordingId),
    enabled: activeTab === "parsed" && !!recording?.response.streaming,
  });

  // Loading state
  if (isLoadingRecording && !recording) {
    return (
      <div className="flex-1 flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="ml-2 text-muted-foreground">Loading recording...</span>
      </div>
    );
  }

  // Not found state
  if (!recording) {
    return (
      <div className="flex-1 flex items-center justify-center py-12">
        <span className="ml-2 text-muted-foreground">Recording not found.</span>
      </div>
    );
  }

  return (
    <div className="w-full h-full flex flex-col bg-background text-foreground">
      <RecordingHeader
        recordingId={recordingId}
        recording={recording}
        sessionContext={sessionContext}
      />

      <div className="flex-1 flex flex-col overflow-hidden bg-background">
        <div className="bg-card border-b">
          <div className="p-6 pb-0">
            <RecordingMetadata recording={recording} />
            {recording.error && <RecordingError error={recording.error} />}
            <RecordingTabs
              activeTab={activeTab}
              onTabChange={setActiveTab}
              tabs={TABS}
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-6 bg-muted/10">
          {activeTab === "request" && <RequestPanel recording={recording} />}
          {activeTab === "response" && <ResponsePanel recording={recording} />}
          {activeTab === "parsed" && (
            <ParsedResponsePanel
              recording={recording}
              parsedData={parsedData}
              isLoading={isParsing}
              error={parseError}
            />
          )}
        </div>
      </div>
    </div>
  );
}
