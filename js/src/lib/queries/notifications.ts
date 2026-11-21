import { useQueryClient } from "@tanstack/react-query";

import { $api } from "@/lib/api/client";
import { linkTokenStaleTime } from "@/lib/queries/connections";
import { invalidateFinanceData } from "@/lib/queries/invalidate";

const notificationsPageSize = 100;

const notificationsRefetchInterval = 5 * 60_000;

export function useNotifications() {
  return $api.useQuery(
    "get",
    "/api/notifications",
    {
      params: { query: { pageSize: notificationsPageSize } },
      headers: { "X-Fake-Lag": "650" },
    },
    {
      refetchOnWindowFocus: true,
      refetchInterval: notificationsRefetchInterval,
    },
  );
}

export function useReauthLinkToken(connectionId: string) {
  return $api.useQuery(
    "post",
    "/api/connections/{connectionId}/reauth/link-token",
    {
      params: { path: { connectionId } },
      headers: { "X-Fake-Lag": "650" },
    },
    { staleTime: linkTokenStaleTime, refetchOnWindowFocus: false },
  );
}

export function useCompleteReauth() {
  const queryClient = useQueryClient();

  return $api.useMutation("post", "/api/connections/{connectionId}/reauth/complete", {
    onSuccess: async () => {
      queryClient.removeQueries({
        queryKey: ["post", "/api/connections/{connectionId}/reauth/link-token"],
      });

      await invalidateFinanceData(queryClient);
    },
  });
}
