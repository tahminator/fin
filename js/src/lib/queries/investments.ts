import { keepPreviousData } from "@tanstack/react-query";

import { $api } from "@/lib/api/client";
import { flattenPages, nextPage, pageSize } from "@/lib/queries/pagination";
import type { InvestmentRange } from "@/lib/types/api";

export interface InvestmentScope {
  accountId?: string | undefined;
  excludeAccountId: string[];
}

function scopeQuery({ accountId, excludeAccountId }: InvestmentScope) {
  return accountId === undefined ? { excludeAccountId } : { accountId, excludeAccountId };
}

export function useInvestmentHoldings(scope: InvestmentScope) {
  return $api.useQuery("get", "/api/investments/holdings", {
    params: { query: scopeQuery(scope) },
  });
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

export function useInvestmentSecurities(scope: InvestmentScope) {
  return $api.useQuery("get", "/api/investments/securities", {
    params: { query: scopeQuery(scope) },
  });
}

export function useInvestmentHistory(
  range: InvestmentRange,
  scope: InvestmentScope,
  securityId: string[] = [],
) {
  return $api.useQuery(
    "get",
    "/api/investments/history",
    { params: { query: { range, securityId, ...scopeQuery(scope) } } },
    { placeholderData: keepPreviousData },
  );
}
