import { useParams } from "react-router";

import { HoldingView } from "@/app/investments/_components/holding-view";
import { accountLink, portfolioLink } from "@/app/investments/_components/investments-search";
import { usePortfolioRange } from "@/app/investments/_hooks/use-portfolio-range";
import { DetailPanel } from "@/components/detail-panel";
import { useDetailTitle } from "@/lib/hooks/use-detail-title";

export function InvestmentHoldingPage() {
  const { accountId, securityId = "" } = useParams();
  const range = usePortfolioRange();
  const title = useDetailTitle("Holding");

  return (
    <DetailPanel
      title={title}
      backLabel={accountId === undefined ? "Investments" : "Account"}
      backTo={accountId === undefined ? portfolioLink(range) : accountLink(range, accountId)}
    >
      <HoldingView range={range} scope={{ accountId, securityId }} />
    </DetailPanel>
  );
}
