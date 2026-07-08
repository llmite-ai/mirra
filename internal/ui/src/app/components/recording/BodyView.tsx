import React, { useMemo, useState } from "react";
import { AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";
import { Recording } from "@/lib/api";
import { RequestView, ResponseView } from "@/lib/conversation";
import {
  isClaudeMessagesBody,
  normalizeClaudeRequest,
  normalizeClaudeResponse,
  reconstructClaudeStream,
} from "@/lib/claude";
import {
  isResponsesRequestBody,
  normalizeResponsesRequest,
  normalizeResponsesResponse,
  reconstructResponsesStream,
} from "@/lib/codex";
import { countEvents, looksBinary, looksLikeSSE, parseSSE } from "@/lib/sse";
import { JsonView } from "@/components/JsonView";
import { RequestOverview } from "@/components/conversation/RequestOverview";
import { ResponseOverview } from "@/components/conversation/ResponseOverview";

type ViewMode = "pretty" | "json" | "raw";

interface Analysis {
  /** Semantic request view, when the body matched a known provider shape */
  request?: RequestView;
  /** Semantic response view, ditto */
  response?: ResponseView;
  /** Parsed JSON for the fallback tree (object bodies, or JSON-parseable strings) */
  json?: unknown;
  /** Raw string body (SSE text, plain text, mangled binary) */
  raw?: string;
  binary: boolean;
}

function isResponsesTraffic(recording: Recording): boolean {
  return (
    recording.provider === "chatgpt" ||
    (recording.provider === "openai" &&
      recording.request.path.startsWith("/v1/responses"))
  );
}

function analyze(
  recording: Recording,
  which: "request" | "response",
): Analysis {
  const body =
    which === "request" ? recording.request.body : recording.response.body;
  const analysis: Analysis = { binary: false };
  if (body == null || body === "") return analysis;

  if (typeof body === "string") {
    analysis.raw = body;
    if (looksBinary(body)) {
      analysis.binary = true;
      return analysis;
    }
    if (looksLikeSSE(body)) {
      if (which === "response") {
        try {
          const events = parseSSE(body);
          if (recording.provider === "claude") {
            analysis.response = reconstructClaudeStream(events) ?? undefined;
          } else if (isResponsesTraffic(recording)) {
            analysis.response = reconstructResponsesStream(events) ?? undefined;
          }
          if (!analysis.response && events.length > 0) {
            // Unknown stream shape: at least summarize the events
            analysis.response = { eventCounts: countEvents(events) };
          }
        } catch {
          // Malformed stream falls back to raw
        }
      }
      return analysis;
    }
    try {
      analysis.json = JSON.parse(body);
    } catch {
      return analysis;
    }
  } else {
    analysis.json = body;
  }

  const json: any = analysis.json;
  try {
    if (which === "request") {
      if (recording.provider === "claude" && isClaudeMessagesBody(json)) {
        analysis.request = normalizeClaudeRequest(json);
      } else if (
        isResponsesTraffic(recording) &&
        isResponsesRequestBody(json)
      ) {
        analysis.request = normalizeResponsesRequest(json);
      }
    } else {
      if (recording.provider === "claude") {
        analysis.response = normalizeClaudeResponse(json) ?? undefined;
      } else if (isResponsesTraffic(recording)) {
        analysis.response = normalizeResponsesResponse(json) ?? undefined;
      }
    }
  } catch {
    // Any surprise in a provider shape falls back to the JSON view
  }

  return analysis;
}

interface BodyViewProps {
  recording: Recording;
  which: "request" | "response";
}

/**
 * Renders a request/response body: a semantic view when the payload matches
 * a known provider shape, with a JSON tree / raw text fallback always a
 * toggle away.
 */
export function BodyView({ recording, which }: BodyViewProps) {
  const analysis = useMemo(() => analyze(recording, which), [recording, which]);

  const hasPretty = analysis.request != null || analysis.response != null;
  const fallback: ViewMode = analysis.json !== undefined ? "json" : "raw";
  const [mode, setMode] = useState<ViewMode>(hasPretty ? "pretty" : fallback);

  if (analysis.raw === undefined && analysis.json === undefined) {
    return <p className="text-sm text-muted-foreground italic">No body</p>;
  }

  const modes: { id: ViewMode; label: string }[] = [];
  if (hasPretty) modes.push({ id: "pretty", label: "Formatted" });
  if (analysis.json !== undefined) modes.push({ id: "json", label: "JSON" });
  if (analysis.raw !== undefined) modes.push({ id: "raw", label: "Raw" });

  const active = modes.some((m) => m.id === mode) ? mode : modes[0].id;

  return (
    <div className="space-y-3">
      {modes.length > 1 && (
        <ModeToggle modes={modes} active={active} onChange={setMode} />
      )}

      {analysis.binary && (
        <div className="flex items-start gap-2 border bg-muted/40 p-2 text-xs text-muted-foreground">
          <AlertTriangle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
          This body is not readable text — it was likely recorded compressed by
          an older version of mirra.
        </div>
      )}

      {active === "pretty" && analysis.request && (
        <RequestOverview view={analysis.request} />
      )}
      {active === "pretty" && analysis.response && (
        <ResponseOverview view={analysis.response} />
      )}
      {active === "json" && (
        <div className="border bg-muted/20 p-3">
          <JsonView data={analysis.json} />
        </div>
      )}
      {active === "raw" && <RawText text={analysis.raw ?? ""} />}
    </div>
  );
}

function ModeToggle({
  modes,
  active,
  onChange,
}: {
  modes: { id: ViewMode; label: string }[];
  active: ViewMode;
  onChange: (mode: ViewMode) => void;
}) {
  return (
    <div className="inline-flex border divide-x">
      {modes.map((m) => (
        <button
          key={m.id}
          onClick={() => onChange(m.id)}
          className={cn(
            "px-3 py-1 text-xs font-medium transition-colors",
            active === m.id
              ? "bg-primary text-primary-foreground"
              : "bg-card text-muted-foreground hover:text-foreground hover:bg-muted/50",
          )}
        >
          {m.label}
        </button>
      ))}
    </div>
  );
}

const RAW_PREVIEW_BYTES = 100_000;

function RawText({ text }: { text: string }) {
  const [showAll, setShowAll] = useState(false);
  const truncated = !showAll && text.length > RAW_PREVIEW_BYTES;
  const shown = truncated ? text.slice(0, RAW_PREVIEW_BYTES) : text;

  return (
    <div>
      <pre className="text-xs bg-muted/20 border p-3 overflow-x-auto whitespace-pre-wrap break-words">
        {shown}
      </pre>
      {truncated && (
        <button
          onClick={() => setShowAll(true)}
          className="mt-1 text-xs text-primary hover:underline"
        >
          Show all ({(text.length / 1024).toFixed(0)} KB)
        </button>
      )}
    </div>
  );
}
