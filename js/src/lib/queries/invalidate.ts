import type { QueryClient } from "@tanstack/react-query";

const financeQueryPaths = [
  "/api/summary",
  "/api/connections",
  "/api/accounts",
  "/api/transactions",
  "/api/notifications",
] as const;

export async function invalidateFinanceData(queryClient: QueryClient): Promise<void> {
  await Promise.all(
    financeQueryPaths.map((path) => queryClient.invalidateQueries({ queryKey: ["get", path] })),
  );
}
