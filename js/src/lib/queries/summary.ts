import { $api } from "@/lib/api/client";

export function useSummary() {
  return $api.useQuery("get", "/api/summary");
}
