import React, { useMemo, useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

/**
 * Dependency-free collapsible JSON tree. Large collections and long strings
 * render truncated with explicit expand affordances so multi-MB bodies never
 * freeze the page.
 */

const MAX_STRING_PREVIEW = 240;
const MAX_CHILDREN = 100;

interface JsonViewProps {
  data: unknown;
  /** Depth expanded on first render; deeper nodes start collapsed */
  defaultExpandDepth?: number;
}

export function JsonView({ data, defaultExpandDepth = 2 }: JsonViewProps) {
  return (
    <div className="font-mono text-xs leading-5 overflow-x-auto">
      <JsonNode value={data} depth={0} expandDepth={defaultExpandDepth} />
    </div>
  );
}

function JsonString({ value }: { value: string }) {
  const [expanded, setExpanded] = useState(false);
  const long = value.length > MAX_STRING_PREVIEW;
  const shown = expanded || !long ? value : value.slice(0, MAX_STRING_PREVIEW);
  return (
    <span className="text-chart-4 whitespace-pre-wrap break-all">
      "{shown}"
      {long && (
        <button
          onClick={() => setExpanded(!expanded)}
          className="ml-1 text-muted-foreground hover:text-foreground underline decoration-dotted"
        >
          {expanded
            ? "less"
            : `+${(value.length - MAX_STRING_PREVIEW).toLocaleString()} chars`}
        </button>
      )}
    </span>
  );
}

function JsonNode({
  value,
  depth,
  expandDepth,
  propertyKey,
}: {
  value: unknown;
  depth: number;
  expandDepth: number;
  propertyKey?: string;
}) {
  const isCollection = value !== null && typeof value === "object";
  const [open, setOpen] = useState(depth < expandDepth);
  const [childLimit, setChildLimit] = useState(MAX_CHILDREN);

  const entries = useMemo<[string, unknown][]>(() => {
    if (!isCollection) return [];
    return Array.isArray(value)
      ? value.map((v, i) => [String(i), v] as [string, unknown])
      : Object.entries(value as Record<string, unknown>);
  }, [isCollection, value]);

  const keyLabel =
    propertyKey !== undefined ? (
      <span className="text-chart-2">{propertyKey}</span>
    ) : null;

  if (!isCollection) {
    return (
      <div style={{ paddingLeft: depth === 0 ? 0 : 14 }}>
        {keyLabel}
        {keyLabel && <span className="text-muted-foreground">: </span>}
        <JsonLeaf value={value} />
      </div>
    );
  }

  const isArray = Array.isArray(value);
  const braces = isArray ? "[]" : "{}";
  const summary = isArray
    ? `${entries.length} items`
    : `${entries.length} keys`;

  return (
    <div style={{ paddingLeft: depth === 0 ? 0 : 14 }}>
      <button
        onClick={() => setOpen(!open)}
        className="inline-flex items-center gap-0.5 hover:bg-muted/60 -ml-0.5 px-0.5"
      >
        {open ? (
          <ChevronDown className="h-3 w-3 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-3 w-3 text-muted-foreground shrink-0" />
        )}
        {keyLabel}
        {keyLabel && <span className="text-muted-foreground">: </span>}
        <span className="text-muted-foreground">
          {open ? braces[0] : `${braces[0]}…${braces[1]}`}
        </span>
        {!open && (
          <span className="text-muted-foreground/70 ml-1">{summary}</span>
        )}
      </button>
      {open && (
        <>
          {entries.slice(0, childLimit).map(([k, v]) => (
            <JsonNode
              key={k}
              value={v}
              depth={depth + 1}
              expandDepth={expandDepth}
              propertyKey={isArray ? undefined : k}
            />
          ))}
          {entries.length > childLimit && (
            <button
              onClick={() => setChildLimit(childLimit + MAX_CHILDREN)}
              className="text-muted-foreground hover:text-foreground underline decoration-dotted"
              style={{ paddingLeft: 14 }}
            >
              show {Math.min(MAX_CHILDREN, entries.length - childLimit)} more of{" "}
              {entries.length - childLimit} remaining
            </button>
          )}
          <div className="text-muted-foreground">{braces[1]}</div>
        </>
      )}
    </div>
  );
}

function JsonLeaf({ value }: { value: unknown }) {
  if (value === null) return <span className="text-chart-5">null</span>;
  switch (typeof value) {
    case "string":
      return <JsonString value={value} />;
    case "number":
      return <span className="text-chart-1">{String(value)}</span>;
    case "boolean":
      return <span className="text-chart-5">{String(value)}</span>;
    case "undefined":
      return <span className="text-muted-foreground">undefined</span>;
    default:
      return <span>{String(value)}</span>;
  }
}
