import React, { useState } from "react";
import { ArrowLeft, Copy, Download, Check } from "lucide-react";
import { format } from "date-fns";
import { useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Recording } from "@/lib/api";
import { RelativeTime } from "@/components/RelativeTime";
import {
  getStatusColor,
  getProviderStyles,
  getProviderLabel,
  getMethodStyles,
} from "@/lib/styles";

interface RecordingHeaderProps {
  recordingId: string;
  recording?: Recording;
}

const CHIP =
  "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium";

/**
 * Compact header: the request's own method + path is the identity, with the
 * key facts (status, provider, model, timing, transport) packed into a single
 * full-width strip beneath it.
 */
export function RecordingHeader({
  recordingId,
  recording,
}: RecordingHeaderProps) {
  const navigate = useNavigate();
  const [copied, setCopied] = useState(false);
  const [idCopied, setIdCopied] = useState(false);

  const handleCopy = async () => {
    if (!recording) return;
    try {
      await navigator.clipboard.writeText(JSON.stringify(recording, null, 2));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error("Failed to copy:", error);
    }
  };

  const handleDownload = () => {
    if (!recording) return;
    try {
      const blob = new Blob([JSON.stringify(recording, null, 2)], {
        type: "application/json",
      });
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

  const handleCopyId = async () => {
    try {
      await navigator.clipboard.writeText(recordingId);
      setIdCopied(true);
      setTimeout(() => setIdCopied(false), 2000);
    } catch (error) {
      console.error("Failed to copy id:", error);
    }
  };

  const body: any = recording?.request.body;
  const model = body && typeof body === "object" ? body.model : undefined;

  return (
    <div className="space-y-2">
      {/* Identity row: back • method • path • status ......... actions */}
      <div className="flex items-center gap-3">
        <button
          onClick={() => navigate("/recordings")}
          className="p-1.5 hover:bg-muted rounded-lg transition-colors shrink-0"
          aria-label="Back to recordings list"
        >
          <ArrowLeft className="h-4 w-4" />
        </button>

        <div className="flex items-center gap-2.5 min-w-0 flex-1">
          {recording && (
            <span
              className={`${CHIP} font-mono uppercase tracking-wide ${getMethodStyles(recording.request.method)}`}
            >
              {recording.request.method}
            </span>
          )}
          <span className="font-mono text-sm truncate">
            {recording?.request.path}
            {recording?.request.query ? `?${recording.request.query}` : ""}
          </span>
          {recording && (
            <span
              className={`${CHIP} font-mono shrink-0 ${getStatusColor(recording.response.status)}`}
            >
              {recording.response.status}
            </span>
          )}
        </div>

        <div className="flex items-center gap-2 shrink-0">
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

      {/* Meta strip: dense, single line, fills the width */}
      {recording && (
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground pl-[2.375rem]">
          <span className={`${CHIP} ${getProviderStyles(recording.provider)}`}>
            {getProviderLabel(recording.provider)}
          </span>
          {model && (
            <span className="font-mono text-foreground/80">{model}</span>
          )}
          <Sep />
          <span>{recording.timing.duration_ms}ms</span>
          {recording.response.streaming && (
            <>
              <Sep />
              <span>SSE stream</span>
            </>
          )}
          <Sep />
          <span>
            {format(new Date(recording.timestamp), "MMM d, yyyy HH:mm:ss")}
          </span>
          <RelativeTime
            timestamp={recording.timestamp}
            className="text-muted-foreground/70"
          />
          <button
            onClick={handleCopyId}
            title={`${recordingId} — click to copy`}
            className="ml-auto inline-flex items-center gap-1 font-mono text-[11px] text-muted-foreground/70 hover:text-foreground transition-colors max-w-[18rem] min-w-0"
          >
            {idCopied ? (
              <Check className="h-3 w-3 shrink-0" />
            ) : (
              <Copy className="h-3 w-3 shrink-0" />
            )}
            <span className="truncate">{recordingId}</span>
          </button>
        </div>
      )}
    </div>
  );
}

/** Hairline dot separator between inline meta facts. */
function Sep() {
  return <span className="text-border select-none">·</span>;
}
