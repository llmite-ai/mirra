import React, { useState } from "react";
import {
  Brain,
  ChevronDown,
  ChevronRight,
  CornerDownRight,
  Wrench,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Block, Message } from "@/lib/conversation";
import { JsonView } from "@/components/JsonView";
import { Expandable } from "./Expandable";

const ROLE_STYLES: Record<string, { border: string; label: string }> = {
  user: { border: "border-l-chart-2", label: "text-chart-2" },
  assistant: { border: "border-l-chart-4", label: "text-chart-4" },
  system: { border: "border-l-chart-5", label: "text-chart-5" },
  developer: { border: "border-l-chart-5", label: "text-chart-5" },
  tool: { border: "border-l-chart-3", label: "text-chart-3" },
};

export function MessageCard({ message }: { message: Message }) {
  const style = ROLE_STYLES[message.role] ?? {
    border: "border-l-border",
    label: "text-muted-foreground",
  };

  return (
    <div className={cn("border-l-2 pl-3 py-1", style.border)}>
      <div
        className={cn(
          "text-[10px] font-semibold uppercase tracking-widest mb-1",
          style.label,
        )}
      >
        {message.role}
      </div>
      <div className="space-y-2">
        {message.blocks.map((block, i) => (
          <BlockView key={i} block={block} />
        ))}
      </div>
    </div>
  );
}

export function BlockView({ block }: { block: Block }) {
  switch (block.kind) {
    case "text":
      return (
        <Expandable>
          <p className="text-sm whitespace-pre-wrap break-words">
            {block.text}
          </p>
        </Expandable>
      );
    case "thinking":
      return <ThinkingBlock text={block.text} />;
    case "tool_call":
      return <ToolCallBlock block={block} />;
    case "tool_result":
      return <ToolResultBlock block={block} />;
    case "image":
      return (
        <div>
          {block.src ? (
            <img
              src={block.src}
              alt={block.mediaType ?? "image"}
              className="max-h-48 max-w-full border"
            />
          ) : (
            <span className="text-xs text-muted-foreground italic">
              image {block.mediaType ? `(${block.mediaType})` : ""}
            </span>
          )}
        </div>
      );
    case "other":
      return (
        <CollapsibleRow
          icon={<CornerDownRight className="h-3.5 w-3.5" />}
          title={block.label}
          titleClassName="text-muted-foreground"
        >
          <JsonView data={block.raw} defaultExpandDepth={1} />
        </CollapsibleRow>
      );
  }
}

function ThinkingBlock({ text }: { text: string }) {
  return (
    <CollapsibleRow
      icon={<Brain className="h-3.5 w-3.5" />}
      title="thinking"
      titleClassName="text-muted-foreground"
      preview={text}
    >
      <Expandable maxHeight={400}>
        <p className="text-sm whitespace-pre-wrap break-words text-muted-foreground italic">
          {text}
        </p>
      </Expandable>
    </CollapsibleRow>
  );
}

function ToolCallBlock({
  block,
}: {
  block: Extract<Block, { kind: "tool_call" }>;
}) {
  const preview =
    typeof block.input === "string" ? block.input : previewJSON(block.input);
  return (
    <CollapsibleRow
      icon={<Wrench className="h-3.5 w-3.5" />}
      title={block.name}
      titleClassName="text-chart-3 font-semibold"
      preview={preview}
    >
      {typeof block.input === "string" ? (
        <Expandable maxHeight={400}>
          <pre className="text-xs whitespace-pre-wrap break-words">
            {block.input}
          </pre>
        </Expandable>
      ) : (
        <JsonView data={block.input} defaultExpandDepth={2} />
      )}
    </CollapsibleRow>
  );
}

function ToolResultBlock({
  block,
}: {
  block: Extract<Block, { kind: "tool_result" }>;
}) {
  return (
    <CollapsibleRow
      icon={<CornerDownRight className="h-3.5 w-3.5" />}
      title={block.isError ? "tool result · error" : "tool result"}
      titleClassName={
        block.isError
          ? "text-destructive-foreground font-semibold"
          : "text-chart-3"
      }
      preview={block.text || previewJSON(block.raw)}
    >
      {block.text ? (
        <Expandable maxHeight={400}>
          <pre
            className={cn(
              "text-xs whitespace-pre-wrap break-words",
              block.isError && "text-destructive-foreground",
            )}
          >
            {block.text}
          </pre>
        </Expandable>
      ) : (
        <JsonView data={block.raw} defaultExpandDepth={1} />
      )}
    </CollapsibleRow>
  );
}

function previewJSON(value: unknown): string {
  if (value == null) return "";
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

interface CollapsibleRowProps {
  icon: React.ReactNode;
  title: string;
  titleClassName?: string;
  /** Single-line preview shown while collapsed */
  preview?: string;
  children: React.ReactNode;
}

function CollapsibleRow({
  icon,
  title,
  titleClassName,
  preview,
  children,
}: CollapsibleRowProps) {
  const [open, setOpen] = useState(false);
  return (
    <div className="border bg-muted/20">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center gap-2 px-2 py-1.5 text-left hover:bg-muted/50 transition-colors min-w-0"
      >
        {open ? (
          <ChevronDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        )}
        <span className="text-muted-foreground shrink-0">{icon}</span>
        <span className={cn("text-xs font-mono shrink-0", titleClassName)}>
          {title}
        </span>
        {!open && preview && (
          <span className="text-xs text-muted-foreground truncate min-w-0">
            {preview}
          </span>
        )}
      </button>
      {open && <div className="px-3 pb-2 pt-1 border-t">{children}</div>}
    </div>
  );
}
