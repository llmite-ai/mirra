/**
 * Centralized style utilities for consistent styling across components
 */

/**
 * Returns Tailwind classes for HTTP status codes
 * - 2xx: green (success)
 * - 4xx: yellow (client error)
 * - 5xx: red (server error)
 * - Other: gray (informational/redirect)
 */
export function getStatusColor(status: number): string {
  if (status >= 200 && status < 300) {
    return "text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20";
  }
  if (status >= 400 && status < 500) {
    return "text-yellow-600 dark:text-yellow-400 bg-yellow-50 dark:bg-yellow-900/20";
  }
  if (status >= 500) {
    return "text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20";
  }
  return "text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-800";
}

/**
 * Returns Tailwind classes for text-only status indicators
 * Used in table views where background colors aren't needed
 */
export function getStatusTextColor(status: number): string {
  if (status >= 200 && status < 300) {
    return "text-green-600 dark:text-green-400";
  }
  if (status >= 400 && status < 500) {
    return "text-yellow-600 dark:text-yellow-400";
  }
  if (status >= 500) {
    return "text-red-600 dark:text-red-400";
  }
  return "text-muted-foreground";
}

/**
 * Provider color mappings for consistent branding
 */
const PROVIDER_STYLES = {
  gemini: "bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-300",
  openai: "bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-300",
  claude: "bg-orange-100 text-orange-800 dark:bg-orange-900/20 dark:text-orange-300",
} as const;

/**
 * Returns Tailwind classes for provider badges
 */
export function getProviderStyles(provider: string): string {
  const normalized = provider.toLowerCase();
  if (normalized in PROVIDER_STYLES) {
    return PROVIDER_STYLES[normalized as keyof typeof PROVIDER_STYLES];
  }
  return "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300";
}

/**
 * Returns color classes based on response size
 * - < 10KB: green (small, efficient)
 * - < 1MB: yellow (moderate)
 * - >= 1MB: red (large)
 */
export function getSizeColor(bytes: number): string {
  const TEN_KB = 10 * 1024;
  const ONE_MB = 1024 * 1024;

  if (bytes < TEN_KB) {
    return "text-green-600 dark:text-green-400";
  }
  if (bytes < ONE_MB) {
    return "text-yellow-600 dark:text-yellow-400";
  }
  return "text-red-600 dark:text-red-400";
}

/**
 * Returns Tailwind classes for session context badges
 * Blue color scheme to distinguish from provider badges
 */
export function getSessionBadgeStyles(): string {
  return "bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-300";
}

/**
 * Returns Tailwind classes for error indicator badges
 * Red color scheme to highlight errors in sessions
 */
export function getErrorBadgeStyles(): string {
  return "bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-300";
}
