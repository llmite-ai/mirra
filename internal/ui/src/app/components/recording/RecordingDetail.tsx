import React from "react";
import { Loader2 } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router";
import { fetchRecording } from "@/lib/api";
import { RecordingHeader } from "./RecordingHeader";
import { RecordingMetadata } from "./RecordingMetadata";
import { RecordingError } from "./RecordingError";
import { RecordingTabs } from "./RecordingTabs";
import { RequestPanel } from "./RequestPanel";
import { ResponsePanel } from "./ResponsePanel";

interface RecordingDetailProps {
  recordingId: string;
}

const TABS = [
  { id: "request", label: "Request" },
  { id: "response", label: "Response" },
];

/**
 * Main orchestrator for recording detail view
 * Handles data fetching and coordinates child components
 */
export default function RecordingDetail({ recordingId }: RecordingDetailProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const tabParam = searchParams.get("tab") || "request";
  // "parsed" was its own tab before streams rendered in the response view
  const activeTab = tabParam === "parsed" ? "response" : tabParam;

  const setActiveTab = (tab: string) => {
    setSearchParams(
      (prev) => {
        const newParams = new URLSearchParams(prev);
        newParams.set("tab", tab);
        return newParams;
      },
      { replace: true },
    );
  };

  const { data: recording, isLoading: isLoadingRecording } = useQuery({
    queryKey: ["recording", recordingId],
    queryFn: () => fetchRecording(recordingId),
    enabled: !!recordingId,
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
      <RecordingHeader recordingId={recordingId} recording={recording} />

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
        </div>
      </div>
    </div>
  );
}
