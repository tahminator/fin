import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

export function useTransactions(accountId: string | undefined) {
  const query = accountId === undefined ? { pageSize } : { pageSize, accountId };

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
