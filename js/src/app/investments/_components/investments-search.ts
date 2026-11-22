import { z } from "zod";

import type { PortfolioRange } from "@/lib/types/api";

export const portfolioRanges = [
  { value: "1h", label: "1H", description: "Past hour" },
  { value: "1d", label: "1D", description: "Past day" },
  { value: "3d", label: "3D", description: "Past 3 days" },
  { value: "7d", label: "1W", description: "Past week" },
  { value: "1mo", label: "1M", description: "Past month" },
  { value: "1y", label: "1Y", description: "Past year" },
  { value: "all", label: "ALL", description: "All time" },
] as const satisfies { value: PortfolioRange; label: string; description: string }[];

const rangeValues = [
  "1h",
  "1d",
  "3d",
  "7d",
  "1mo",
  "1y",
  "all",
] as const satisfies PortfolioRange[];

export const investmentsSearchSchema = z.object({
  range: z.enum(rangeValues).default("1d"),
  accountId: z.uuid().optional(),
  securityId: z.uuid().optional(),
});

export type InvestmentsSearch = z.infer<typeof investmentsSearchSchema>;

export function investmentsLink(search: InvestmentsSearch) {
  const params = new URLSearchParams({ range: search.range });

  if (search.accountId !== undefined) {
    params.set("accountId", search.accountId);
  }

  if (search.securityId !== undefined) {
    params.set("securityId", search.securityId);
  }

  return { pathname: "/investments", search: params.toString() };
}

export function rangeDescription(range: PortfolioRange): string {
  return portfolioRanges.find((option) => option.value === range)?.description ?? "";
}

export function isIntraday(range: PortfolioRange): boolean {
  return range !== "1y" && range !== "all";
}
