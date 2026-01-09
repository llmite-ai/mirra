import React from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { format } from "date-fns";
import { Loader2 } from "lucide-react";
import { fetchSessionGroup } from "@/lib/api";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../ui/table";
import { getStatusTextColor, getProviderStyles } from "@/lib/styles";
import { truncateId } from "@/lib/formatters";

interface SessionGroupExpandedProps {
  traceId: string;
  onRecordingClick?: (id: string) => void;
}

export default function SessionGroupExpanded({
  traceId,
  onRecordingClick,
}: SessionGroupExpandedProps) {
  const navigate = useNavigate();

  const { data, isLoading, error } = useQuery({
    queryKey: ["session-group", traceId],
    queryFn: () => fetchSessionGroup(traceId),
    // Only fetch when this component is mounted (session is expanded)
    staleTime: 5000, // Consider data fresh for 5 seconds
  });

  const handleRecordingClick = (id: string) => {
    if (onRecordingClick) {
      onRecordingClick(id);
    } else {
      navigate(`/recordings/${id}`);
    }
  };

  if (isLoading) {
    return (
      <div className="p-8 flex items-center justify-center bg-card/50">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md text-red-800 dark:text-red-300 text-sm">
        Error loading session details: {(error as Error).message}
      </div>
    );
  }

  if (!data || !data.recordings || data.recordings.length === 0) {
    return (
      <div className="p-8 text-center text-muted-foreground text-sm">
        No recordings found in this session
      </div>
    );
  }

  return (
    <div className="mt-2 ml-8 mr-2 border rounded-md bg-card/50 overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>ID</TableHead>
            <TableHead>Time</TableHead>
            <TableHead>Provider</TableHead>
            <TableHead>Method</TableHead>
            <TableHead>Path</TableHead>
            <TableHead>Status</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.recordings.map((recording) => (
            <TableRow
              key={recording.id}
              onClick={() => handleRecordingClick(recording.id)}
              className="cursor-pointer"
            >
              <TableCell className="font-mono text-xs text-muted-foreground">
                {truncateId(recording.id)}
              </TableCell>
              <TableCell className="text-sm text-foreground">
                {format(new Date(recording.timestamp), "HH:mm:ss")}
              </TableCell>
              <TableCell>
                <span
                  className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${getProviderStyles(recording.provider)}`}
                >
                  {recording.provider}
                </span>
              </TableCell>
              <TableCell className="text-sm font-mono text-foreground">
                {recording.method}
              </TableCell>
              <TableCell className="text-sm font-mono max-w-xs truncate text-foreground">
                {recording.path}
              </TableCell>
              <TableCell
                className={`font-medium ${getStatusTextColor(recording.status)}`}
              >
                {recording.status}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
