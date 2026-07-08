import React, { useState } from "react";
import { Copy, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Recording } from "@/lib/api";
import { formatJSON } from "@/lib/formatters";
import { BodyView } from "./BodyView";
import { HeadersSection } from "./HeadersSection";

interface RequestPanelProps {
  recording: Recording;
}

/**
 * Displays request details: endpoint, headers, body
 */
export function RequestPanel({ recording }: RequestPanelProps) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = () => {
    const requestData = formatJSON({
      method: recording.request.method,
      path: recording.request.path,
      query: recording.request.query,
      headers: recording.request.headers,
      body: recording.request.body,
    });
    navigator.clipboard.writeText(requestData);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-card border rounded-md">
      <div className="flex items-center justify-between p-3 bg-muted/30 border-b">
        <h3 className="font-semibold">Request</h3>
        <Button size="sm" variant="ghost" onClick={copyToClipboard}>
          {copied ? (
            <>
              <Check className="h-4 w-4 mr-1" />
              Copied
            </>
          ) : (
            <>
              <Copy className="h-4 w-4 mr-1" />
              Copy
            </>
          )}
        </Button>
      </div>
      <div className="p-4 space-y-3">
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Endpoint
          </label>
          <p className="text-sm mt-1 font-mono">
            {recording.request.method} {recording.request.path}
            {recording.request.query && `?${recording.request.query}`}
          </p>
        </div>

        <HeadersSection headers={recording.request.headers} />

        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Body
          </label>
          <div className="mt-1">
            <BodyView recording={recording} which="request" />
          </div>
        </div>
      </div>
    </div>
  );
}
