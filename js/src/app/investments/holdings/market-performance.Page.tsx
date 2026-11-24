import { useParams } from "react-router";

import { holdingLink } from "@/app/investments/_components/investments-search";
import { MarketPerformanceView } from "@/app/investments/_components/market-performance-view";
import { usePortfolioRange } from "@/app/investments/_hooks/use-portfolio-range";
import { DetailPanel } from "@/components/detail-panel";
import { useDetailTitle } from "@/lib/hooks/use-detail-title";

export function InvestmentMarketPerformancePage() {
  const { accountId, securityId = "" } = useParams();
  const range = usePortfolioRange();
  const title = useDetailTitle("Market performance");

  return (
    <DetailPanel
      title={title}
      backLabel="Holding"
      backTo={holdingLink(range, securityId, accountId)}
    >
      <MarketPerformanceView range={range} scope={{ accountId, securityId }} />
    </DetailPanel>
  );
}
