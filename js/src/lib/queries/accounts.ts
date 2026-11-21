import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

export function useAccounts() {
  return $api.useInfiniteQuery(
    "get",
    "/api/accounts",
    { params: { query: { pageSize } }, headers: { "X-Fake-Lag": "900" } },
    {
      pageParamName: "page",
      initialPageParam: 1,
      getNextPageParam: nextPage,
      select: (data) => flattenPages(data),
    },
  );
}

export function useConnectionAccounts(connectionId: string) {
  return $api.useInfiniteQuery(
    "get",
    "/api/accounts",
    { params: { query: { pageSize, connectionId } }, headers: { "X-Fake-Lag": "800" } },
    {
      pageParamName: "page",
      initialPageParam: 1,
      getNextPageParam: nextPage,
      select: (data) => flattenPages(data),
    },
  );
}

export function useAllAccounts() {
  return $api.useQuery(
    "get",
    "/api/accounts",
    { params: { query: { pageSize: 100 } }, headers: { "X-Fake-Lag": "850" } },
    { select: (data) => data.items },
  );
}
