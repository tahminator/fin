import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

export function useInvestmentHoldings(
  accountId: string | undefined,
  excludeAccountId: string[] = [],
) {
  const query = accountId === undefined ? { excludeAccountId } : { accountId, excludeAccountId };

  return $api.useQuery("get", "/api/investments/holdings", { params: { query } });
}

export function useInvestmentTransactions(
  accountId: string | undefined,
  excludeAccountId: string[] = [],
) {
  const query =
    accountId === undefined
      ? { pageSize, excludeAccountId }
      : { pageSize, accountId, excludeAccountId };

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
