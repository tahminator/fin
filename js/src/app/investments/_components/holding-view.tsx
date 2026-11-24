import { InvestmentActivity } from "@/app/investments/_components/investment-activity";
import { MarketDataNotice } from "@/app/investments/_components/market-data-notice";
import { PortfolioSummary } from "@/app/investments/_components/portfolio-summary";
import { PositionDetails } from "@/app/investments/_components/position-details";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { usePortfolioPerformance } from "@/lib/queries/investments";
import type { InvestmentScope } from "@/lib/queries/investments";
import type { PortfolioRange } from "@/lib/types/api";

export function HoldingView({ range, scope }: { range: PortfolioRange; scope: InvestmentScope }) {
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
    return <EmptyState title="Nothing held here" description="Try a longer time range." />;
  }

  return (
    <>
      <MarketDataNotice status={data.marketData} />
      <PortfolioSummary
        title={position.securityType === "cash" ? "Cash" : (position.name ?? "Holding")}
        performance={data}
      />
      <PositionDetails
        position={position}
        currency={data.currency}
        range={range}
        accountId={scope.accountId}
      />
      <InvestmentActivity scope={scope} />
    </>
  );
}
