import { useQueryClient } from "@tanstack/react-query";

import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

export function useAuthorizationRequest(requestId: string) {
  return $api.useQuery(
    "get",
    "/api/oauth/requests/{requestId}",
    { params: { path: { requestId } } },
    { retry: false, refetchOnWindowFocus: false },
  );
}

export function useApproveAuthorization() {
  return $api.useMutation("post", "/api/oauth/requests/{requestId}/approve");
}

export function useDenyAuthorization() {
  return $api.useMutation("post", "/api/oauth/requests/{requestId}/deny");
}

export function useGrants() {
  return $api.useInfiniteQuery(
    "get",
    "/api/oauth/grants",
    { params: { query: { pageSize } } },
    {
      pageParamName: "page",
      initialPageParam: 1,
      getNextPageParam: nextPage,
      select: (data) => flattenPages(data),
    },
  );
}

export function useRevokeGrant() {
  const queryClient = useQueryClient();

  return $api.useMutation("delete", "/api/oauth/grants/{grantId}", {
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["get", "/api/oauth/grants"] }),
  });
}
