import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

export function useTransactions(accountId: string | undefined, excludeAccountId: string[] = []) {
  const query =
    accountId === undefined
      ? { pageSize, excludeAccountId }
      : { pageSize, accountId, excludeAccountId };

  return $api.useInfiniteQuery(
    "get",
    "/api/transactions",
    { params: { query } },
    {
      pageParamName: "page",
      initialPageParam: 1,
      getNextPageParam: nextPage,
      select: (data) => flattenPages(data),
    },
  );
}
