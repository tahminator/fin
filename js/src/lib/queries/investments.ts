import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

export function useInvestmentHoldings(accountId: string | undefined) {
  return $api.useQuery(
    "get",
    "/api/investments/holdings",
    accountId === undefined ? undefined : { params: { query: { accountId } } },
  );
}

export function useInvestmentTransactions(accountId: string | undefined) {
  const query = accountId === undefined ? { pageSize } : { pageSize, accountId };

  return $api.useInfiniteQuery(
    "get",
    "/api/investments/transactions",
    { params: { query } },
    {
      pageParamName: "page",
      initialPageParam: 1,
      getNextPageParam: nextPage,
      select: (data) => flattenPages(data),
    },
  );
}
