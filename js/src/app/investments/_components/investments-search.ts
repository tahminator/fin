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

const defaultRange: PortfolioRange = "1d";

export function parseRange(value: string | null): PortfolioRange {
  return portfolioRanges.find((option) => option.value === value)?.value ?? defaultRange;
}

export function rangeDescription(range: PortfolioRange): string {
  return portfolioRanges.find((option) => option.value === range)?.description ?? "";
}

export function isIntraday(range: PortfolioRange): boolean {
  return range !== "1y" && range !== "all";
}

function withRange(pathname: string, range: PortfolioRange) {
  return { pathname, search: new URLSearchParams({ range }).toString() };
}

export function portfolioLink(range: PortfolioRange) {
  return withRange("/investments", range);
}

export function accountLink(range: PortfolioRange, accountId: string) {
  return withRange(`/investments/accounts/${accountId}`, range);
}

export function holdingLink(range: PortfolioRange, securityId: string, accountId?: string) {
  return withRange(holdingPath(securityId, accountId), range);
}

export function marketPerformanceLink(
  range: PortfolioRange,
  securityId: string,
  accountId?: string,
) {
  return withRange(`${holdingPath(securityId, accountId)}/market-performance`, range);
}

function holdingPath(securityId: string, accountId?: string) {
  return accountId === undefined
    ? `/investments/holdings/${securityId}`
    : `/investments/accounts/${accountId}/holdings/${securityId}`;
}
