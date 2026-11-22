import { InvestmentActivity } from "@/app/investments/_components/investment-activity";
import { MarketDataNotice } from "@/app/investments/_components/market-data-notice";
import { PortfolioSummary } from "@/app/investments/_components/portfolio-summary";
import { PositionList } from "@/app/investments/_components/position-list";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { usePortfolioPerformance } from "@/lib/queries/investments";
import type { PortfolioRange } from "@/lib/types/api";

export function AccountView({ range, accountId }: { range: PortfolioRange; accountId: string }) {
  const performance = usePortfolioPerformance(range, { accountId });

  if (performance.isPending) {
    return <LoadingState />;
  }

  if (performance.isError) {
    return <ErrorState error={performance.error} onRetry={performance.refetch} />;
  }

  const { data } = performance;

  if (data.positions.length === 0) {
    return <EmptyState title="Nothing held here" description="Try a longer time range." />;
  }

  return (
    <>
      <MarketDataNotice status={data.marketData} />
      <PortfolioSummary title="Account value" performance={data} />
      <PositionList
        positions={data.positions}
        currency={data.currency}
        range={range}
        accountId={accountId}
      />
      <InvestmentActivity scope={{ accountId }} />
    </>
  );
}
