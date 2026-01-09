import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { Button } from "../components/ui/button";
import { fetchSessionGroups } from "../lib/api";
import SessionFilters from "../components/sessions/SessionFilters";
import SessionGroupRow from "../components/sessions/SessionGroupRow";
import SessionGroupExpanded from "../components/sessions/SessionGroupExpanded";

export default function SessionsView() {
  const [page, setPage] = useState(1);
  const [limit] = useState(50);
  const [provider, setProvider] = useState("");
  const [hasErrors, setHasErrors] = useState(false);
  const [search, setSearch] = useState("");
  const [expandedSessionId, setExpandedSessionId] = useState<string | null>(null);

  // Fetch session groups with auto-refresh every 10 seconds
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["session-groups", page, limit, provider, hasErrors || undefined, search],
    queryFn: () =>
      fetchSessionGroups(
        page,
        limit,
        provider || undefined,
        hasErrors || undefined,
        undefined,
        undefined,
      ),
    refetchInterval: 10000, // Auto-refresh every 10 seconds
    refetchIntervalInBackground: true,
  });

  const handleClearFilters = () => {
    setProvider("");
    setHasErrors(false);
    setSearch("");
    setPage(1);
  };

  const handleRefresh = () => {
    refetch();
  };

  const handleToggleExpand = (traceId: string) => {
    // Accordion logic: collapse if already expanded, otherwise expand
    setExpandedSessionId(expandedSessionId === traceId ? null : traceId);
  };

  return (
    <div className="space-y-4 w-full">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-foreground">Session Groups</h2>
          <p className="text-sm text-muted-foreground">
            {data?.total ? `${data.total} total sessions` : "Loading..."}
          </p>
        </div>
        <Button
          onClick={handleRefresh}
          disabled={isFetching}
          className="flex items-center gap-2"
        >
          <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>

      {/* Filters */}
      <SessionFilters
        provider={provider}
        hasErrors={hasErrors}
        search={search}
        onProviderChange={(value) => {
          setProvider(value);
          setPage(1);
        }}
        onHasErrorsChange={(value) => {
          setHasErrors(value);
          setPage(1);
        }}
        onSearchChange={(value) => {
          setSearch(value);
          setPage(1);
        }}
        onClearFilters={handleClearFilters}
      />

      {/* Error State */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md text-red-800 dark:text-red-300">
          Error loading sessions: {(error as Error).message}
        </div>
      )}

      {/* Loading State */}
      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <div className="text-muted-foreground">Loading sessions...</div>
        </div>
      )}

      {/* Session Groups List */}
      {!isLoading && data && (
        <>
          <div className="space-y-2">
            {data.groups.length === 0 ? (
              <div className="p-8 text-center text-muted-foreground border rounded-md bg-card">
                No session groups found
              </div>
            ) : (
              data.groups.map((group) => {
                const traceId = group.trace_id || group.session_id;
                const isExpanded = expandedSessionId === traceId;

                return (
                  <div key={traceId}>
                    <SessionGroupRow
                      group={group}
                      isExpanded={isExpanded}
                      onToggle={() => handleToggleExpand(traceId)}
                    />
                    {isExpanded && <SessionGroupExpanded traceId={traceId} />}
                  </div>
                );
              })
            )}
          </div>

          {/* Pagination */}
          {data.groups.length > 0 && (
            <div className="flex items-center justify-between">
              <div className="text-sm text-muted-foreground">
                Page {data.page} of {Math.ceil(data.total / data.limit)}
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
  );
}
