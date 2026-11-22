import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

const excludeType: "investment"[] = ["investment"];

export function useCashAccounts() {
  return $api.useInfiniteQuery(
    "get",
    "/api/accounts",
    { params: { query: { pageSize, excludeType } } },
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
    { params: { query: { pageSize, connectionId } } },
    {
      pageParamName: "page",
      initialPageParam: 1,
      getNextPageParam: nextPage,
      select: (data) => flattenPages(data),
    },
  );
}

export function useAllCashAccounts() {
  return $api.useQuery(
    "get",
    "/api/accounts",
    { params: { query: { pageSize: 100, excludeType } } },
    { select: (data) => data.items },
  );
}
