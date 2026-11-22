import { keepPreviousData } from "@tanstack/react-query";

import { $api } from "@/lib/api/client";

export function useSummary(excludeAccountId: string[] = []) {
  return $api.useQuery(
    "get",
    "/api/summary",
    { params: { query: { excludeAccountId } } },
    { placeholderData: keepPreviousData },
  );
}

const excludeAccountType: "investment"[] = ["investment"];

export function useCashSummary(excludeAccountId: string[]) {
  return $api.useQuery(
    "get",
    "/api/summary",
    { params: { query: { excludeAccountId, excludeAccountType } } },
    { placeholderData: keepPreviousData },
  );
}

export function useCashFlow(excludeAccountId: string[] = []) {
  return $api.useQuery(
    "get",
    "/api/summary/cash-flow",
    { params: { query: { excludeAccountId, excludeAccountType } } },
    { placeholderData: keepPreviousData },
  );
}
