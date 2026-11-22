import { useParams } from "react-router";

import { DetailPanel } from "@/app/investments/_components/detail-panel";
import { HoldingView } from "@/app/investments/_components/holding-view";
import { accountLink, portfolioLink } from "@/app/investments/_components/investments-search";
import { useDetailTitle } from "@/app/investments/_hooks/use-detail-title";
import { usePortfolioRange } from "@/app/investments/_hooks/use-portfolio-range";

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
