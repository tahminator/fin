import { $api } from "@/lib/api/client";

export function useVersion() {
  return $api.useQuery("get", "/version", undefined, { staleTime: "static" });
}
