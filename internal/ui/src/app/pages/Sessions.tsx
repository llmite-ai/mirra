import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { format } from "date-fns";
import { AlertTriangle, RefreshCw } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import { Button } from "../components/ui/button";
import { fetchSessionGroups, GroupingDisabledError } from "../lib/api";
import { getProviderStyles, getProviderLabel } from "@/lib/styles";
import { truncateId } from "@/lib/formatters";
import { formatTimespan } from "@/lib/sessions";
import { RelativeTime } from "@/components/RelativeTime";

export default function Sessions() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [limit] = useState(50);

  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["sessions", page, limit],
    queryFn: () => fetchSessionGroups(page, limit),
    refetchInterval: 10000,
    refetchIntervalInBackground: true,
    retry: (count, err) => !(err instanceof GroupingDisabledError) && count < 3,
  });

  return (
    <div className="mx-auto flex-1">
      <div className="space-y-4 w-full p-4">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight text-foreground">
              Sessions
            </h1>
            <p className="text-sm text-muted-foreground">
              {data?.total
                ? `${data.total} total sessions`
                : "Recordings grouped by agent session"}
            </p>
          </div>
          <Button
            variant="outline"
            onClick={() => refetch()}
            disabled={isFetching}
            className="flex items-center gap-2"
          >
            <RefreshCw
              className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`}
            />
            Refresh
          </Button>
        </div>

        {/* Error State */}
        {error instanceof GroupingDisabledError ? (
          <div className="p-8 text-center text-muted-foreground border rounded-lg bg-card shadow-elev">
            Session grouping is not enabled on this server. Enable recording to
            group traffic into sessions.
          </div>
        ) : error ? (
          <div className="p-4 rounded-lg bg-rose-500/10 border border-rose-500/25 text-rose-700 dark:text-rose-300">
            Error loading sessions: {(error as Error).message}
          </div>
        ) : null}

        {/* Loading State */}
        {isLoading && !error && (
          <div className="flex items-center justify-center py-12">
            <div className="text-muted-foreground">Loading sessions...</div>
          </div>
        )}

        {/* Table */}
        {!isLoading && data && (
          <>
            <div className="border rounded-lg bg-card shadow-elev overflow-hidden [&_th]:text-[11px] [&_th]:uppercase [&_th]:tracking-[0.08em] [&_th]:font-semibold [&_th]:text-muted-foreground">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Trace</TableHead>
                    <TableHead>Started</TableHead>
                    <TableHead>Span</TableHead>
                    <TableHead>Requests</TableHead>
                    <TableHead>Providers</TableHead>
                    <TableHead>Errors</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.groups.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={6}
                        className="text-center py-8 text-muted-foreground"
                      >
                        No sessions yet. Sessions appear as traffic with trace
                        or session metadata flows through the proxy.
                      </TableCell>
                    </TableRow>
                  ) : (
                    data.groups.map((group) => (
                      <TableRow
                        key={group.trace_id}
                        onClick={() => navigate(`/sessions/${group.trace_id}`)}
                        className="cursor-pointer"
                      >
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {truncateId(group.trace_id, 12)}
                        </TableCell>
                        <TableCell className="text-sm">
                          <div>
                            {format(
                              new Date(group.first_timestamp),
                              "MMM d, HH:mm:ss",
                            )}
                          </div>
                          <RelativeTime
                            timestamp={group.first_timestamp}
                            className="text-xs text-muted-foreground"
                          />
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {formatTimespan(
                            group.first_timestamp,
                            group.last_timestamp,
                          )}
                        </TableCell>
                        <TableCell className="text-sm">
                          {group.request_count}
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-1">
                            {group.providers.map((provider) => (
                              <span
                                key={provider}
                                className={
                                  "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium " +
                                  getProviderStyles(provider)
                                }
                              >
                                {getProviderLabel(provider)}
                              </span>
                            ))}
                          </div>
                        </TableCell>
                        <TableCell>
                          {group.has_errors && (
                            <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400" />
                          )}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>

            {/* Pagination */}
            {data.groups.length > 0 && (
              <div className="flex items-center justify-between">
                <div className="text-sm text-muted-foreground">
                  Page {data.page} of{" "}
                  {Math.max(1, Math.ceil(data.total / data.limit))}
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    onClick={() => setPage(page - 1)}
                    disabled={page === 1}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setPage(page + 1)}
                    disabled={!data.hasMore}
                  >
                    Next
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
