import React from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router";
import { format } from "date-fns";
import { AlertTriangle, ArrowLeft } from "lucide-react";
import { RecordingsTable } from "../components/RecordingsTable";
import { fetchSessionGroup } from "../lib/api";
import { formatTimespan } from "@/lib/sessions";

export default function Session() {
  const { traceId } = useParams<{ traceId: string }>();
  const navigate = useNavigate();

  const { data, isLoading, error } = useQuery({
    queryKey: ["session", traceId],
    queryFn: () => fetchSessionGroup(traceId!),
    enabled: !!traceId,
    refetchInterval: 10000,
  });

  if (!traceId) {
    navigate("/sessions");
    return null;
  }

  return (
    <div className="mx-auto flex-1">
      <div className="space-y-4 w-full p-4">
        {/* Header */}
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate("/sessions")}
            className="p-2 hover:bg-muted rounded-md transition-colors"
            aria-label="Back to sessions list"
          >
            <ArrowLeft className="h-5 w-5" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-foreground">Session</h1>
            <p className="text-sm text-muted-foreground font-mono">{traceId}</p>
          </div>
        </div>

        {error && (
          <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md text-red-800 dark:text-red-300">
            Error loading session: {(error as Error).message}
          </div>
        )}

        {isLoading && (
          <div className="flex items-center justify-center py-12">
            <div className="text-muted-foreground">Loading session...</div>
          </div>
        )}

        {data && (
          <>
            <div className="flex flex-wrap gap-x-8 gap-y-3 border rounded-md bg-card p-4">
              <SessionField label="Started">
                {format(
                  new Date(data.group.first_timestamp),
                  "MMM d, yyyy HH:mm:ss",
                )}
              </SessionField>
              <SessionField label="Span">
                {formatTimespan(
                  data.group.first_timestamp,
                  data.group.last_timestamp,
                )}
              </SessionField>
              <SessionField label="Requests">
                {data.group.request_count}
              </SessionField>
              {data.group.session_id && (
                <SessionField label="Session ID">
                  <span className="font-mono text-xs">
                    {data.group.session_id}
                  </span>
                </SessionField>
              )}
              {data.group.has_errors && (
                <SessionField label="Errors">
                  <span className="inline-flex items-center gap-1 text-red-600 dark:text-red-400">
                    <AlertTriangle className="h-4 w-4" /> yes
                  </span>
                </SessionField>
              )}
            </div>

            <RecordingsTable
              recordings={data.recordings}
              onSelect={(recording) =>
                navigate(`/recordings/${recording.id}?session=${traceId}`)
              }
              emptyMessage="No recordings in this session"
            />
          </>
        )}
      </div>
    </div>
  );
}

function SessionField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <label className="text-sm font-medium text-muted-foreground">
        {label}
      </label>
      <p className="text-sm mt-1">{children}</p>
    </div>
  );
}
