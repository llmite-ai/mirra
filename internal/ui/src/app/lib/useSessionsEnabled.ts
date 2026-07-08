import { useQuery } from "@tanstack/react-query";
import { fetchSessionGroups, GroupingDisabledError } from "./api";

/**
 * Feature-detects session grouping (the server answers 501 when it is off)
 * so navigation can hide the Sessions page. Optimistically true until the
 * server says otherwise.
 */
export function useSessionsEnabled(): boolean {
  const { data } = useQuery({
    queryKey: ["sessions-enabled"],
    queryFn: async () => {
      try {
        await fetchSessionGroups(1, 1);
        return true;
      } catch (err) {
        if (err instanceof GroupingDisabledError) return false;
        return true; // transient errors shouldn't hide the page
      }
    },
    staleTime: Infinity,
    retry: false,
  });
  return data ?? true;
}
