import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

export function useAccounts() {
  return $api.useInfiniteQuery(
    "get",
    "/api/accounts",
    { params: { query: { pageSize } } },
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
    { params: { query: { pageSize: 100 } } },
    { select: (data) => data.items },
  );
}
