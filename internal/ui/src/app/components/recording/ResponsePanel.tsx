import React, { useState } from "react";
import { Copy, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Recording } from "@/lib/api";
import { formatJSON } from "@/lib/formatters";
import { BodyView } from "./BodyView";
import { HeadersSection } from "./HeadersSection";

interface ResponsePanelProps {
  recording: Recording;
}

/**
 * Displays response details: status, headers, body
 */
export function ResponsePanel({ recording }: ResponsePanelProps) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = () => {
    const responseData = formatJSON({
      status: recording.response.status,
      headers: recording.response.headers,
      body: recording.response.body,
    });
    navigator.clipboard.writeText(responseData);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-card border rounded-lg shadow-elev overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-border/60">
        <h3 className="font-semibold tracking-tight">Response</h3>
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
        <HeadersSection headers={recording.response.headers} />

        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Body
          </label>
          <div className="mt-1">
            <BodyView recording={recording} which="response" />
          </div>
        </div>
      </div>
    </div>
  );
}
