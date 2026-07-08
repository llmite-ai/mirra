import React, { useEffect, useRef, useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useSearchParams } from "react-router";
import { format } from "date-fns";
import { Loader2, X } from "lucide-react";
import { fetchRecordings, fetchSessionGroup } from "../lib/api";
import {
  getStatusTextColor,
  getProviderStyles,
  getProviderLabel,
} from "@/lib/styles";
import { truncateId } from "@/lib/formatters";

interface RecordingSidebarProps {
  currentRecordingId: string;
  /** When set, the sidebar lists this session's recordings instead of the latest traffic */
  sessionId?: string;
}

export default function RecordingSidebar({
  currentRecordingId,
  sessionId,
}: RecordingSidebarProps) {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(320);
  const [isResizing, setIsResizing] = useState(false);

  const startResizing = useCallback(() => {
    setIsResizing(true);
  }, []);

  const stopResizing = useCallback(() => {
    setIsResizing(false);
  }, []);

  const resize = useCallback((mouseMoveEvent: MouseEvent) => {
    const newWidth = mouseMoveEvent.clientX;
    if (newWidth > 200 && newWidth < 800) {
      setWidth(newWidth);
    }
  }, []);

  useEffect(() => {
    if (isResizing) {
      window.addEventListener("mousemove", resize);
      window.addEventListener("mouseup", stopResizing);
    }
    return () => {
      window.removeEventListener("mousemove", resize);
      window.removeEventListener("mouseup", stopResizing);
    };
  }, [isResizing, resize, stopResizing]);

  const { data, isLoading } = useQuery({
    queryKey: ["recordings", "sidebar"],
    queryFn: () => fetchRecordings(1, 100), // Fetch first 100 for the sidebar
    refetchInterval: 5000,
    enabled: !sessionId,
  });

  const { data: sessionData, isLoading: isLoadingSession } = useQuery({
    queryKey: ["session", sessionId],
    queryFn: () => fetchSessionGroup(sessionId!),
    refetchInterval: 5000,
    enabled: !!sessionId,
  });

  const recordings =
    (sessionId ? sessionData?.recordings : data?.recordings) || [];

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!recordings.length) return;

      const currentIndex = recordings.findIndex(
        (r) => r.id === currentRecordingId,
      );
      if (currentIndex === -1) return;

      const search = searchParams.toString();
      const searchWithQuestionMark = search ? `?${search}` : "";

      if (e.key === "ArrowUp" || e.key === "ArrowLeft") {
        e.preventDefault();
        if (currentIndex > 0) {
          navigate({
            pathname: `/recordings/${recordings[currentIndex - 1].id}`,
            search: searchWithQuestionMark,
          });
        }
      } else if (e.key === "ArrowDown" || e.key === "ArrowRight") {
        e.preventDefault();
        if (currentIndex < recordings.length - 1) {
          navigate({
            pathname: `/recordings/${recordings[currentIndex + 1].id}`,
            search: searchWithQuestionMark,
          });
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [currentRecordingId, recordings, navigate, searchParams]);

  // Scroll to active item
  useEffect(() => {
    if (scrollRef.current) {
      const activeElement =
        scrollRef.current.querySelector(`[data-active="true"]`);
      if (activeElement) {
        activeElement.scrollIntoView({ block: "nearest", behavior: "smooth" });
      }
    }
  }, [currentRecordingId, recordings]);

  if (isLoading || isLoadingSession) {
    return (
      <div className="w-80 border-r bg-muted/10 flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div
      className="relative border-r bg-muted/10 flex flex-col h-full flex-shrink-0"
      style={{ width }}
    >
      <div className="p-4 border-b border-border/60 bg-background/50 backdrop-blur flex items-center justify-between gap-2">
        <h3 className="font-semibold text-[11px] text-muted-foreground uppercase tracking-[0.08em] truncate">
          {sessionId
            ? `Session ${truncateId(sessionId, 12)}`
            : "Recent Recordings"}
        </h3>
        {sessionId && (
          <button
            onClick={() => navigate(`/recordings/${currentRecordingId}`)}
            className="p-1 hover:bg-muted rounded transition-colors shrink-0"
            aria-label="Leave session view"
            title="Show recent recordings instead"
          >
            <X className="h-4 w-4 text-muted-foreground" />
          </button>
        )}
      </div>
      <div className="flex-1 overflow-y-auto" ref={scrollRef}>
        {recordings.length === 0 ? (
          <div className="p-4 text-center text-muted-foreground text-sm">
            No recordings found
          </div>
        ) : (
          <div className="divide-y">
            {recordings.map((recording) => {
              const isActive = recording.id === currentRecordingId;
              return (
                <button
                  key={recording.id}
                  data-active={isActive}
                  onClick={() => {
                    const search = searchParams.toString();
                    navigate({
                      pathname: `/recordings/${recording.id}`,
                      search: search ? `?${search}` : "",
                    });
                  }}
                  className={`w-full text-left p-3 transition-colors focus:outline-none ${
                    isActive
                      ? "bg-primary/10 border-l-2 border-l-primary"
                      : "border-l-2 border-l-transparent hover:bg-muted/40"
                  }`}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span
                      className={`text-xs font-medium ${getStatusTextColor(recording.status)}`}
                    >
                      {recording.status} {recording.method}
                    </span>
                    <span className="text-[10px] text-muted-foreground">
                      {format(new Date(recording.timestamp), "HH:mm:ss")}
                    </span>
                  </div>
                  <div
                    className="text-xs font-mono truncate text-foreground/80 mb-1"
                    title={recording.path}
                  >
                    {recording.path}
                  </div>
                  <div className="flex items-center justify-between text-[10px] text-muted-foreground">
                    <span
                      className={
                        getProviderStyles(recording.provider) +
                        " px-1.5 py-0.5 rounded-full font-medium"
                      }
                    >
                      {getProviderLabel(recording.provider)}
                    </span>
                    <span>{recording.duration}ms</span>
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>
      <div
        className="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/50 active:bg-primary transition-colors z-10"
        onMouseDown={startResizing}
      />
    </div>
  );
}
