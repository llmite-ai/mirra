import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { Search, RefreshCw } from "lucide-react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { RecordingsTable } from "../components/RecordingsTable";
import { fetchRecordings } from "../lib/api";

export default function Recordings() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [limit] = useState(50);
  const [provider, setProvider] = useState("");
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");

  // Fetch recordings list with auto-refresh every 10 seconds
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["recordings", page, limit, provider, search],
    queryFn: () => fetchRecordings(page, limit, provider, search),
    refetchInterval: 10000, // Auto-refresh every 10 seconds
    refetchIntervalInBackground: true,
  });

  const handleSearch = () => {
    setSearch(searchInput);
    setPage(1);
  };

  const handleClearFilters = () => {
    setProvider("");
    setSearch("");
    setSearchInput("");
    setPage(1);
  };

  const handleRefresh = () => {
    refetch();
  };

  return (
    <div className="mx-auto flex-1">
      <div className="space-y-4 w-full p-4">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-foreground">Recordings</h1>
            <p className="text-sm text-muted-foreground">
              {data?.total ? `${data.total} total recordings` : "Loading..."}
            </p>
          </div>
          <Button
            onClick={handleRefresh}
            disabled={isFetching}
            className="flex items-center gap-2"
          >
            <RefreshCw
              className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`}
            />
            Refresh
          </Button>
        </div>

        {/* Filters */}
        <div className="flex gap-4 items-end">
          <div className="flex-1">
            <label className="text-sm font-medium mb-1 block text-muted-foreground">
              Search
            </label>
            <div className="flex gap-2">
              <Input
                placeholder="Search by ID, path, or error..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    handleSearch();
                  }
                }}
              />
              <Button
                onClick={handleSearch}
                className="flex items-center gap-2"
              >
                <Search className="h-4 w-4" />
                Search
              </Button>
            </div>
          </div>
          <div className="w-48">
            <label className="text-sm font-medium mb-1 block text-muted-foreground">
              Provider
            </label>
            <select
              value={provider}
              onChange={(e) => {
                setProvider(e.target.value);
                setPage(1);
              }}
              className="w-full px-3 py-2 border rounded-md bg-background text-foreground border-input"
            >
              <option value="">All Providers</option>
              <option value="claude">Claude</option>
              <option value="chatgpt">Codex</option>
              <option value="openai">OpenAI</option>
              <option value="gemini">Gemini</option>
            </select>
          </div>
          {(provider || search) && (
            <Button variant="outline" onClick={handleClearFilters}>
              Clear Filters
            </Button>
          )}
        </div>

        {/* Error State */}
        {error && (
          <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md text-red-800 dark:text-red-300">
            Error loading recordings: {(error as Error).message}
          </div>
        )}

        {/* Loading State */}
        {isLoading && (
          <div className="flex items-center justify-center py-12">
            <div className="text-muted-foreground">Loading recordings...</div>
          </div>
        )}

        {/* Table */}
        {!isLoading && data && (
          <>
            <RecordingsTable
              recordings={data.recordings}
              onSelect={(recording) => navigate(`/recordings/${recording.id}`)}
            />

            {/* Pagination */}
            {data.recordings.length > 0 && (
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
    </div>
  );
}
