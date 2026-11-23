import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";

export function useTransactions(accountId: string | undefined, excludeAccountId: string[] = []) {
  const excludeAccountType: "investment"[] = ["investment"];
  const query =
    accountId === undefined
      ? { pageSize, excludeAccountId, excludeAccountType }
      : { pageSize, accountId, excludeAccountId, excludeAccountType };

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

export function useTransaction(transactionId: string) {
  return $api.useQuery("get", "/api/transactions/{transactionId}", {
    params: { path: { transactionId } },
  });
}
