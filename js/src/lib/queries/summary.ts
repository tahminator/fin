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

export function useCashFlow(excludeAccountId: string[] = []) {
  return $api.useQuery(
    "get",
    "/api/summary/cash-flow",
    { params: { query: { excludeAccountId } } },
    { placeholderData: keepPreviousData },
  );
}
