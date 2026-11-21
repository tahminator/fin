import { useQueryClient } from "@tanstack/react-query";

import { $api } from "@/lib/api/client";
import { invalidateFinanceData } from "@/lib/queries/invalidate";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

export const linkTokenStaleTime = 30 * 60_000;

export function useConnections() {
  return $api.useInfiniteQuery(
    "get",
    "/api/connections",
    { params: { query: { pageSize } }, headers: { "X-Fake-Lag": "850" } },
    {
      pageParamName: "page",
      initialPageParam: 1,
      getNextPageParam: nextPage,
      select: (data) => flattenPages(data),
    },
  );
}

export function useLinkToken() {
  return $api.useQuery(
    "post",
    "/api/connections/link-token",
    { headers: { "X-Fake-Lag": "650" } },
    {
      staleTime: linkTokenStaleTime,
      gcTime: linkTokenStaleTime,
      refetchOnWindowFocus: false,
    },
  );
}

export function useLinkConnection() {
  const queryClient = useQueryClient();

  return $api.useMutation("post", "/api/connections", {
    onSuccess: async () => {
      queryClient.removeQueries({ queryKey: ["post", "/api/connections/link-token"] });

      await invalidateFinanceData(queryClient);
    },
  });
}

export function useSyncConnection() {
  const queryClient = useQueryClient();

  return $api.useMutation("post", "/api/connections/{connectionId}/sync", {
    onSettled: () => invalidateFinanceData(queryClient),
  });
}

export function useSyncAll() {
  const queryClient = useQueryClient();

  return $api.useMutation("post", "/api/connections/sync", {
    onSettled: () => invalidateFinanceData(queryClient),
  });
}

export function useRemoveConnection() {
  const queryClient = useQueryClient();

  return $api.useMutation("delete", "/api/connections/{connectionId}", {
    onSuccess: () => invalidateFinanceData(queryClient),
  });
}

export function useCreateAccountSelectionLinkToken() {
  return $api.useMutation("post", "/api/connections/{connectionId}/account-selection/link-token");
}

export function useCompleteAccountSelection() {
  const queryClient = useQueryClient();

  return $api.useMutation("post", "/api/connections/{connectionId}/account-selection/complete", {
    onSuccess: () => invalidateFinanceData(queryClient),
  });
}
