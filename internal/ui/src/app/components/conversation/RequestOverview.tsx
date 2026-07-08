import React, { useState } from "react";
import { ChevronDown, ChevronRight, ScrollText, Wrench } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { RequestView } from "@/lib/conversation";
import { JsonView } from "@/components/JsonView";
import { Expandable } from "./Expandable";
import { MessageCard } from "./MessageCard";

/**
 * Semantic view of a normalized request: model + params, system prompt,
 * tool definitions, and the conversation itself.
 */
export function RequestOverview({ view }: { view: RequestView }) {
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        {view.model && <Badge variant="secondary">{view.model}</Badge>}
        {view.stream && <Badge variant="outline">stream</Badge>}
        <ParamChips params={view.params} />
      </div>

      {view.system && <SystemPromptSection text={view.system} />}
      {view.tools.length > 0 && <ToolsSection tools={view.tools} />}

      <div>
        <SectionLabel>
          Messages{" "}
          <span className="text-muted-foreground">
            ({view.messages.length})
          </span>
        </SectionLabel>
        <div className="space-y-3">
          {view.messages.map((message, i) => (
            <MessageCard key={i} message={message} />
          ))}
        </div>
      </div>
    </div>
  );
}

export function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground mb-2">
      {children}
    </div>
  );
}

function ParamChips({ params }: { params: Record<string, unknown> }) {
  const [showObjects, setShowObjects] = useState<string | null>(null);

  const entries = Object.entries(params);
  if (entries.length === 0) return null;

  return (
    <>
      {entries.map(([key, value]) => {
        const scalar = value === null || typeof value !== "object";
        if (scalar) {
          return (
            <Badge
              key={key}
              variant="outline"
              className="font-mono font-normal"
            >
              <span>
                <span className="text-muted-foreground">{key}=</span>
                {String(value)}
              </span>
            </Badge>
          );
        }
        return (
          <React.Fragment key={key}>
            <Badge
              variant="outline"
              className="font-mono font-normal cursor-pointer hover:bg-accent"
              onClick={() => setShowObjects(showObjects === key ? null : key)}
            >
              <span className="text-muted-foreground">{key}</span>
              {showObjects === key ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronRight className="h-3 w-3" />
              )}
            </Badge>
            {showObjects === key && (
              <div className="w-full rounded-lg border bg-muted/20 p-2">
                <JsonView data={value} defaultExpandDepth={2} />
              </div>
            )}
          </React.Fragment>
        );
      })}
    </>
  );
}

function SystemPromptSection({ text }: { text: string }) {
  const [open, setOpen] = useState(false);
  const firstLine = text.split("\n", 1)[0];
  return (
    <div className="rounded-lg border bg-muted/20 overflow-hidden">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center gap-2 px-2 py-1.5 text-left hover:bg-muted/50 transition-colors min-w-0"
      >
        {open ? (
          <ChevronDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        )}
        <ScrollText className="h-3.5 w-3.5 text-chart-5 shrink-0" />
        <span className="text-xs font-mono text-chart-5 font-semibold shrink-0">
          system
        </span>
        <span className="text-xs text-muted-foreground shrink-0">
          {text.length.toLocaleString()} chars
        </span>
        {!open && (
          <span className="text-xs text-muted-foreground truncate min-w-0">
            {firstLine}
          </span>
        )}
      </button>
      {open && (
        <div className="px-3 pb-2 pt-1 border-t">
          <Expandable maxHeight={500}>
            <p className="text-sm whitespace-pre-wrap break-words">{text}</p>
          </Expandable>
        </div>
      )}
    </div>
  );
}

function ToolsSection({ tools }: { tools: RequestView["tools"] }) {
  const [open, setOpen] = useState(false);
  const [openTool, setOpenTool] = useState<string | null>(null);

  return (
    <div className="rounded-lg border bg-muted/20 overflow-hidden">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center gap-2 px-2 py-1.5 text-left hover:bg-muted/50 transition-colors min-w-0"
      >
        {open ? (
          <ChevronDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        )}
        <Wrench className="h-3.5 w-3.5 text-chart-3 shrink-0" />
        <span className="text-xs font-mono text-chart-3 font-semibold shrink-0">
          tools ({tools.length})
        </span>
        {!open && (
          <span className="text-xs text-muted-foreground truncate min-w-0">
            {tools.map((t) => t.name).join(", ")}
          </span>
        )}
      </button>
      {open && (
        <div className="border-t divide-y">
          {tools.map((tool) => (
            <div key={tool.name} className="px-3 py-1.5">
              <button
                onClick={() =>
                  setOpenTool(openTool === tool.name ? null : tool.name)
                }
                className="flex w-full items-baseline gap-2 text-left min-w-0"
              >
                <span className="text-xs font-mono font-semibold shrink-0">
                  {tool.name}
                </span>
                <span className="text-xs text-muted-foreground truncate min-w-0">
                  {tool.description?.split("\n", 1)[0]}
                </span>
              </button>
              {openTool === tool.name && (
                <div className="mt-2 space-y-2">
                  {tool.description && (
                    <Expandable maxHeight={200}>
                      <p className="text-xs whitespace-pre-wrap text-muted-foreground">
                        {tool.description}
                      </p>
                    </Expandable>
                  )}
                  {tool.schema != null && (
                    <JsonView data={tool.schema} defaultExpandDepth={2} />
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
