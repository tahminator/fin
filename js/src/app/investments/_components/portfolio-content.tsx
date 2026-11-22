import { AccountPerformanceList } from "@/app/investments/_components/account-performance-list";
import { InvestmentActivity } from "@/app/investments/_components/investment-activity";
import type { InvestmentsSearch } from "@/app/investments/_components/investments-search";
import { MarketDataNotice } from "@/app/investments/_components/market-data-notice";
import { PortfolioSummary } from "@/app/investments/_components/portfolio-summary";
import { PositionDetails } from "@/app/investments/_components/position-details";
import { PositionList } from "@/app/investments/_components/position-list";
import type { PortfolioPerformance } from "@/lib/types/api";

function scopeTitle(performance: PortfolioPerformance, search: InvestmentsSearch): string {
  const [position] = performance.positions;
  const [account] = performance.accounts;

  if (search.securityId !== undefined) {
    return position?.ticker ?? position?.name ?? "Holding";
  }

  if (search.accountId !== undefined) {
    return account?.name ?? "Account";
  }

  return "Portfolio";
}

interface PortfolioContentProps {
  performance: PortfolioPerformance;
  search: InvestmentsSearch;
}

export function PortfolioContent({ performance, search }: PortfolioContentProps) {
  const [firstPosition] = performance.positions;
  const position = search.securityId === undefined ? undefined : firstPosition;
  const unscoped = search.accountId === undefined && search.securityId === undefined;

  return (
    <>
      <MarketDataNotice status={performance.marketData} />
      <PortfolioSummary
        title={scopeTitle(performance, search)}
        performance={performance}
        search={search}
      />
      {position === undefined ? (
        <PositionList
          positions={performance.positions}
          currency={performance.currency}
          search={search}
        />
      ) : (
        <PositionDetails position={position} currency={performance.currency} />
      )}
      {unscoped ? (
        <AccountPerformanceList
          accounts={performance.accounts}
          currency={performance.currency}
          search={search}
        />
      ) : null}
      <InvestmentActivity scope={{ accountId: search.accountId, securityId: search.securityId }} />
    </>
  );
}
