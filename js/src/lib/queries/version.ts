import { $api } from "@/lib/api/client";

export function useVersion() {
  return $api.useQuery(
    "get",
    "/version",
    { headers: { "X-Fake-Lag": "600" } },
    { staleTime: "static" },
  );
}
