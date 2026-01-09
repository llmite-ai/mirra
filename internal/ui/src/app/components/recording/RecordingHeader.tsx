import React, { useState } from "react";
import { ArrowLeft, Copy, Download, Check, FolderTree } from "lucide-react";
import { useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Recording } from "@/lib/api";
import { getSessionBadgeStyles } from "@/lib/styles";
import { formatTraceId } from "@/lib/formatters";

interface SessionContext {
  traceId: string;
  position: number;
  total: number;
}

interface RecordingHeaderProps {
  recordingId: string;
  recording?: Recording;
  sessionContext?: SessionContext;
}

/**
 * Header bar with back button and recording ID
 */
export function RecordingHeader({ recordingId, recording, sessionContext }: RecordingHeaderProps) {
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
      console.error('Failed to copy:', error);
    }
  };

  const handleDownload = () => {
    if (!recording) return;

    try {
      const recordingJson = JSON.stringify(recording, null, 2);
      const blob = new Blob([recordingJson], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `recording-${recording.id}.json`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to download:', error);
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
          {sessionContext && (
            <button
              onClick={() => navigate(`/recordings?view=sessions&expanded=${sessionContext.traceId}`)}
              className={`inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium mt-2 ${getSessionBadgeStyles()} hover:opacity-80 transition-opacity`}
            >
              <FolderTree className="h-3 w-3" />
              Part of Session {formatTraceId(sessionContext.traceId, 12)} (Request {sessionContext.position} of {sessionContext.total})
            </button>
          )}
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
