import React, { useState } from "react";
import { Loader2, Copy, Check } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router";
import { fetchRecording, Recording } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { formatJSON } from "@/lib/formatters";
import { RecordingHeader } from "./RecordingHeader";
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
    <div className="w-full h-full flex flex-col overflow-hidden bg-background text-foreground">
      <div className="bg-card border-b border-border/60">
        <div className="px-6 pt-4">
          <RecordingHeader recordingId={recordingId} recording={recording} />
          {recording.error && (
            <div className="mt-4">
              <RecordingError error={recording.error} />
            </div>
          )}
          <div className="mt-4">
            <RecordingTabs
              activeTab={activeTab}
              onTabChange={setActiveTab}
              tabs={TABS}
              actions={
                <CopyPayloadButton
                  recording={recording}
                  activeTab={activeTab}
                />
              }
            />
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6 bg-muted/10">
        {activeTab === "request" && <RequestPanel recording={recording} />}
        {activeTab === "response" && <ResponsePanel recording={recording} />}
      </div>
    </div>
  );
}

/** Copies the payload of whichever tab is currently active. */
function CopyPayloadButton({
  recording,
  activeTab,
}: {
  recording: Recording;
  activeTab: string;
}) {
  const [copied, setCopied] = useState(false);
  const isResponse = activeTab === "response";

  const handleCopy = () => {
    const payload = isResponse
      ? {
          status: recording.response.status,
          headers: recording.response.headers,
          body: recording.response.body,
        }
      : {
          method: recording.request.method,
          path: recording.request.path,
          query: recording.request.query,
          headers: recording.request.headers,
          body: recording.request.body,
        };
    navigator.clipboard.writeText(formatJSON(payload));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Button size="sm" variant="ghost" onClick={handleCopy}>
      {copied ? (
        <>
          <Check className="h-4 w-4 mr-1" />
          Copied
        </>
      ) : (
        <>
          <Copy className="h-4 w-4 mr-1" />
          Copy {isResponse ? "response" : "request"}
        </>
      )}
    </Button>
  );
}
