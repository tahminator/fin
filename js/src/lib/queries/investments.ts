import { keepPreviousData } from "@tanstack/react-query";

import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";
import type { PortfolioRange } from "@/lib/types/api";

export interface InvestmentScope {
  accountId?: string | undefined;
  securityId?: string | undefined;
}

function scopeQuery({ accountId, securityId }: InvestmentScope) {
  return {
    ...(accountId === undefined ? {} : { accountId }),
    ...(securityId === undefined ? {} : { securityId }),
  };
}

export function usePortfolioPerformance(range: PortfolioRange, scope: InvestmentScope) {
  return $api.useQuery(
    "get",
    "/api/investments/performance",
    { params: { query: { range, ...scopeQuery(scope) } } },
    { placeholderData: keepPreviousData, refetchInterval: 30_000 },
  );
}

export function useInvestmentTransactions(scope: InvestmentScope) {
  return $api.useInfiniteQuery(
    "get",
    "/api/investments/transactions",
    { params: { query: { pageSize, ...scopeQuery(scope) } } },
    {
      pageParamName: "page",
      initialPageParam: 1,
      getNextPageParam: nextPage,
      select: (data) => flattenPages(data),
    },
  );
}

export function useInvestmentTransaction(transactionId: string) {
  return $api.useQuery("get", "/api/investments/transactions/{transactionId}", {
    params: { path: { transactionId } },
  });
}
