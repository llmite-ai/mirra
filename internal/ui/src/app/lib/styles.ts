/**
 * Centralized style utilities for consistent styling across components.
 * Badges are soft tints with hairline inset rings; pair with a rounded-full
 * pill container at the call site.
 */

/**
 * Returns Tailwind classes for HTTP status codes
 * - 2xx: emerald (success)
 * - 4xx: amber (client error)
 * - 5xx: rose (server error)
 * - Other: neutral (informational/redirect)
 */
export function getStatusColor(status: number): string {
  if (status >= 200 && status < 300) {
    return "text-emerald-700 dark:text-emerald-300 bg-emerald-500/10 ring-1 ring-inset ring-emerald-500/25";
  }
  if (status >= 400 && status < 500) {
    return "text-amber-700 dark:text-amber-300 bg-amber-500/10 ring-1 ring-inset ring-amber-500/25";
  }
  if (status >= 500) {
    return "text-rose-700 dark:text-rose-300 bg-rose-500/10 ring-1 ring-inset ring-rose-500/25";
  }
  return "text-muted-foreground bg-muted ring-1 ring-inset ring-border";
}

/**
 * Returns Tailwind classes for HTTP method chips, tinted by verb so the
 * request's identity reads at a glance.
 */
export function getMethodStyles(method: string): string {
  switch (method.toUpperCase()) {
    case "GET":
      return "text-sky-700 dark:text-sky-300 bg-sky-500/10 ring-1 ring-inset ring-sky-500/25";
    case "POST":
      return "text-emerald-700 dark:text-emerald-300 bg-emerald-500/10 ring-1 ring-inset ring-emerald-500/25";
    case "PUT":
    case "PATCH":
      return "text-amber-700 dark:text-amber-300 bg-amber-500/10 ring-1 ring-inset ring-amber-500/25";
    case "DELETE":
      return "text-rose-700 dark:text-rose-300 bg-rose-500/10 ring-1 ring-inset ring-rose-500/25";
    default:
      return "text-muted-foreground bg-muted ring-1 ring-inset ring-border";
  }
}

/**
 * Returns Tailwind classes for text-only status indicators
 * Used in table views where background colors aren't needed
 */
export function getStatusTextColor(status: number): string {
  if (status >= 200 && status < 300) {
    return "text-emerald-600 dark:text-emerald-400";
  }
  if (status >= 400 && status < 500) {
    return "text-amber-600 dark:text-amber-400";
  }
  if (status >= 500) {
    return "text-rose-600 dark:text-rose-400";
  }
  return "text-muted-foreground";
}

/**
 * Provider color mappings for consistent branding
 */
const PROVIDER_STYLES = {
  gemini:
    "bg-sky-500/10 text-sky-700 dark:text-sky-300 ring-1 ring-inset ring-sky-500/25",
  openai:
    "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 ring-1 ring-inset ring-emerald-500/25",
  claude:
    "bg-orange-500/10 text-orange-700 dark:text-orange-300 ring-1 ring-inset ring-orange-500/25",
  chatgpt:
    "bg-violet-500/10 text-violet-700 dark:text-violet-300 ring-1 ring-inset ring-violet-500/25",
  unknown: "bg-muted text-muted-foreground ring-1 ring-inset ring-border",
} as const;

/**
 * Display names for providers. "chatgpt" is Codex traffic authenticated
 * with a ChatGPT subscription.
 */
const PROVIDER_LABELS: Record<string, string> = {
  gemini: "Gemini",
  openai: "OpenAI",
  claude: "Claude",
  chatgpt: "Codex",
  unknown: "Unknown",
};

/**
 * Returns Tailwind classes for provider badges
 */
export function getProviderStyles(provider: string): string {
  const normalized = provider.toLowerCase();
  if (normalized in PROVIDER_STYLES) {
    return PROVIDER_STYLES[normalized as keyof typeof PROVIDER_STYLES];
  }
  return PROVIDER_STYLES.unknown;
}

/**
 * Returns the human-readable name for a provider
 */
export function getProviderLabel(provider: string): string {
  return PROVIDER_LABELS[provider.toLowerCase()] ?? provider;
}

/**
 * Returns color classes based on response size
 * - < 10KB: emerald (small, efficient)
 * - < 1MB: amber (moderate)
 * - >= 1MB: rose (large)
 */
export function getSizeColor(bytes: number): string {
  const TEN_KB = 10 * 1024;
  const ONE_MB = 1024 * 1024;

  if (bytes < TEN_KB) {
    return "text-emerald-600 dark:text-emerald-400";
  }
  if (bytes < ONE_MB) {
    return "text-amber-600 dark:text-amber-400";
  }
  return "text-rose-600 dark:text-rose-400";
}
