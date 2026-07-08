import React, { useState } from "react";
import { ArrowLeft, Copy, Download, Check } from "lucide-react";
import { useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Recording } from "@/lib/api";

interface RecordingHeaderProps {
  recordingId: string;
  recording?: Recording;
}

/**
 * Header bar with back button and recording ID
 */
export function RecordingHeader({
  recordingId,
  recording,
}: RecordingHeaderProps) {
  const navigate = useNavigate();
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (!recording) return;

    try {
      const recordingJson = JSON.stringify(recording, null, 2);
      await navigator.clipboard.writeText(recordingJson);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error("Failed to copy:", error);
    }
  };

  const handleDownload = () => {
    if (!recording) return;

    try {
      const recordingJson = JSON.stringify(recording, null, 2);
      const blob = new Blob([recordingJson], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `recording-${recording.id}.json`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Failed to download:", error);
    }
  };

  return (
    <div className="flex items-center justify-between p-6 border-b bg-card">
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate("/recordings")}
          className="p-2 hover:bg-muted rounded-md transition-colors"
          aria-label="Back to recordings list"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div>
          <h2 className="text-xl font-bold">Recording Details</h2>
          <p className="text-sm text-muted-foreground font-mono mt-1">
            {recordingId}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button
          size="sm"
          variant="outline"
          onClick={handleCopy}
          disabled={!recording}
        >
          {copied ? (
            <>
              <Check className="h-4 w-4 mr-1" />
              Copied
            </>
          ) : (
            <>
              <Copy className="h-4 w-4 mr-1" />
              Copy JSON
            </>
          )}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={handleDownload}
          disabled={!recording}
        >
          <Download className="h-4 w-4 mr-1" />
          Download
        </Button>
      </div>
    </div>
  );
}
