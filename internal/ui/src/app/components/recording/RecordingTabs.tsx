import React from "react";
import { cn } from "@/lib/utils";

interface RecordingTabsProps {
  activeTab: string;
  onTabChange: (tab: string) => void;
  tabs: Array<{ id: string; label: string }>;
}

/**
 * Tab navigation with a crisp underline on the active tab.
 */
export function RecordingTabs({
  activeTab,
  onTabChange,
  tabs,
}: RecordingTabsProps) {
  return (
    <div className="flex gap-6 border-b border-border/60">
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
  );
}
