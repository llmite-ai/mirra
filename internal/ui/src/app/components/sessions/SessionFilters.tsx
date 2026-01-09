import React from "react";
import { Search, X } from "lucide-react";
import { Button } from "../ui/button";
import { Input } from "../ui/input";

interface SessionFiltersProps {
  provider: string;
  hasErrors: boolean;
  search: string;
  onProviderChange: (provider: string) => void;
  onHasErrorsChange: (hasErrors: boolean) => void;
  onSearchChange: (search: string) => void;
  onClearFilters: () => void;
}

export default function SessionFilters({
  provider,
  hasErrors,
  search,
  onProviderChange,
  onHasErrorsChange,
  onSearchChange,
  onClearFilters,
}: SessionFiltersProps) {
  const [searchInput, setSearchInput] = React.useState(search);

  const handleSearch = () => {
    onSearchChange(searchInput);
  };

  const hasActiveFilters = provider || hasErrors || search;

  return (
    <div className="flex gap-4 items-end">
      <div className="flex-1">
        <label className="text-sm font-medium mb-1 block text-muted-foreground">
          Search
        </label>
        <div className="flex gap-2">
          <Input
            placeholder="Search by trace ID or session ID..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                handleSearch();
              }
            }}
          />
          <Button onClick={handleSearch} className="flex items-center gap-2">
            <Search className="h-4 w-4" />
            Search
          </Button>
        </div>
      </div>
      <div className="w-48">
        <label className="text-sm font-medium mb-1 block text-muted-foreground">
          Provider
        </label>
        <select
          value={provider}
          onChange={(e) => onProviderChange(e.target.value)}
          className="w-full px-3 py-2 border rounded-md bg-background text-foreground border-input"
        >
          <option value="">All Providers</option>
          <option value="openai">OpenAI</option>
          <option value="claude">Claude</option>
          <option value="gemini">Gemini</option>
        </select>
      </div>
      <div className="flex items-center">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={hasErrors}
            onChange={(e) => onHasErrorsChange(e.target.checked)}
            className="w-4 h-4 rounded border-input"
          />
          <span className="text-sm text-foreground">Show only errors</span>
        </label>
      </div>
      {hasActiveFilters && (
        <Button
          variant="outline"
          onClick={onClearFilters}
          className="flex items-center gap-2"
        >
          <X className="h-4 w-4" />
          Clear Filters
        </Button>
      )}
    </div>
  );
}
