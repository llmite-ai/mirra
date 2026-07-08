import React from "react";
import { cn } from "@/lib/utils";

interface RecordingTabsProps {
  activeTab: string;
  onTabChange: (tab: string) => void;
  tabs: Array<{ id: string; label: string }>;
  /** Optional controls rendered flush-right on the tab bar. */
  actions?: React.ReactNode;
}

/**
 * Tab navigation with a crisp underline on the active tab, and an optional
 * right-aligned action slot sharing the same baseline.
 */
export function RecordingTabs({
  activeTab,
  onTabChange,
  tabs,
  actions,
}: RecordingTabsProps) {
  return (
    <div className="flex items-end justify-between border-b border-border/60">
      <div className="flex gap-6">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
            className={cn(
              "relative pb-3 text-sm font-medium transition-colors",
              activeTab === tab.id
                ? "text-foreground"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            {tab.label}
            {activeTab === tab.id && (
              <span className="absolute inset-x-0 -bottom-px h-0.5 rounded-full bg-primary" />
            )}
          </button>
        ))}
      </div>
      {actions && <div className="pb-1.5">{actions}</div>}
    </div>
  );
}
