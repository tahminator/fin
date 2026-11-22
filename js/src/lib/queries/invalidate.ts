import type { QueryClient } from "@tanstack/react-query";

const financeQueryPaths = [
  "/api/summary",
  "/api/summary/cash-flow",
  "/api/connections",
  "/api/accounts",
  "/api/transactions",
  "/api/investments/holdings",
  "/api/investments/transactions",
  "/api/investments/securities",
  "/api/investments/history",
  "/api/notifications",
] as const;

export async function invalidateFinanceData(queryClient: QueryClient): Promise<void> {
  await Promise.all(
    financeQueryPaths.map((path) => queryClient.invalidateQueries({ queryKey: ["get", path] })),
  );
}
