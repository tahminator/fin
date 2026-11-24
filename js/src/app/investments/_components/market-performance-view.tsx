import { MarketDataNotice } from "@/app/investments/_components/market-data-notice";
import { MarketPerformanceSummary } from "@/app/investments/_components/portfolio-summary";
import { RangePicker } from "@/app/investments/_components/range-picker";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { usePortfolioPerformance } from "@/lib/queries/investments";
import type { InvestmentScope } from "@/lib/queries/investments";
import type { PortfolioRange } from "@/lib/types/api";

export function MarketPerformanceView({
  range,
  scope,
}: {
  range: PortfolioRange;
  scope: InvestmentScope;
}) {
  const performance = usePortfolioPerformance(range, scope);

  if (performance.isPending) {
    return <LoadingState />;
  }

  if (performance.isError) {
    return <ErrorState error={performance.error} onRetry={performance.refetch} />;
  }

  const { data } = performance;
  const [position] = data.positions;

  if (position === undefined) {
    return (
      <>
        <EmptyState title="Nothing held here" description="Try a longer time range." />
        <RangePicker range={range} />
      </>
    );
  }

  const points = position.marketPricePoints;
  if (position.marketPriceChange === null || points.length < 2) {
    return (
      <>
        <MarketDataNotice status={data.marketData} />
        <EmptyState
          title="Market history unavailable"
          description="There isn't enough historical market pricing for this security and range."
        />
        <RangePicker range={range} />
      </>
    );
  }

  return (
    <>
      <MarketDataNotice status={data.marketData} />
      <MarketPerformanceSummary
        points={points}
        range={range}
        currency={data.currency}
        change={position.marketPriceChange}
        changePercent={position.marketPriceChangePercent}
      />
    </>
  );
}
