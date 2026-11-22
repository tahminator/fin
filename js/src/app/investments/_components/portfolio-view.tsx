import { AccountPerformanceList } from "@/app/investments/_components/account-performance-list";
import { EmptyPortfolio } from "@/app/investments/_components/empty-portfolio";
import { InvestmentActivity } from "@/app/investments/_components/investment-activity";
import { MarketDataNotice } from "@/app/investments/_components/market-data-notice";
import { PortfolioSummary } from "@/app/investments/_components/portfolio-summary";
import { PositionList } from "@/app/investments/_components/position-list";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { usePortfolioPerformance } from "@/lib/queries/investments";
import type { PortfolioRange } from "@/lib/types/api";

export function PortfolioView({ range }: { range: PortfolioRange }) {
  const performance = usePortfolioPerformance(range, {});

  if (performance.isPending) {
    return <LoadingState />;
  }

  if (performance.isError) {
    return <ErrorState error={performance.error} onRetry={performance.refetch} />;
  }

  const { data } = performance;

  if (data.positions.length === 0) {
    return <EmptyPortfolio />;
  }

  return (
    <>
      <MarketDataNotice status={data.marketData} />
      <PortfolioSummary title="Portfolio" performance={data} />
      <PositionList positions={data.positions} currency={data.currency} range={range} />
      <AccountPerformanceList accounts={data.accounts} currency={data.currency} range={range} />
      <InvestmentActivity scope={{}} />
    </>
  );
}
