import React from "react";
import { format } from "date-fns";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { RecordingSummary } from "@/lib/api";
import {
  getStatusTextColor,
  getProviderStyles,
  getProviderLabel,
  getSizeColor,
} from "@/lib/styles";
import { formatBytes, truncateId } from "@/lib/formatters";

interface RecordingsTableProps {
  recordings: RecordingSummary[];
  onSelect: (recording: RecordingSummary) => void;
  emptyMessage?: string;
}

/**
 * Recordings listing shared by the recordings page and the session detail.
 */
export function RecordingsTable({
  recordings,
  onSelect,
  emptyMessage = "No recordings found",
}: RecordingsTableProps) {
  return (
    <div className="border rounded-md bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>ID</TableHead>
            <TableHead>Timestamp</TableHead>
            <TableHead>Provider</TableHead>
            <TableHead>Method</TableHead>
            <TableHead>Path</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Duration</TableHead>
            <TableHead>Size</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {recordings.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={8}
                className="text-center py-8 text-muted-foreground"
              >
                {emptyMessage}
              </TableCell>
            </TableRow>
          ) : (
            recordings.map((recording) => (
              <TableRow
                key={recording.id}
                onClick={() => onSelect(recording)}
                className="cursor-pointer"
              >
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {truncateId(recording.id)}
                </TableCell>
                <TableCell className="text-sm text-foreground">
                  {format(new Date(recording.timestamp), "MMM d, HH:mm:ss")}
                </TableCell>
                <TableCell>
                  <span
                    className={
                      "inline-flex items-center px-2 py-1 rounded text-xs font-medium " +
                      getProviderStyles(recording.provider)
                    }
                  >
                    {getProviderLabel(recording.provider)}
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
                <TableCell className="text-sm text-foreground">
                  {recording.duration}ms
                </TableCell>
                <TableCell
                  className={`text-sm font-mono ${getSizeColor(recording.responseSize)}`}
                >
                  {formatBytes(recording.responseSize)}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}
