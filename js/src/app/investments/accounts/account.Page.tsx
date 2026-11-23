import { Outlet, useParams } from "react-router";

import { AccountView } from "@/app/investments/_components/account-view";
import { portfolioLink } from "@/app/investments/_components/investments-search";
import { usePortfolioRange } from "@/app/investments/_hooks/use-portfolio-range";
import { DetailPanel } from "@/components/detail-panel";
import { useDetailTitle } from "@/lib/hooks/use-detail-title";

export function InvestmentAccountPage() {
  const { accountId = "" } = useParams();
  const range = usePortfolioRange();
  const title = useDetailTitle("Account");

  return (
    <>
      <DetailPanel title={title} backLabel="Investments" backTo={portfolioLink(range)}>
        <AccountView range={range} accountId={accountId} />
      </DetailPanel>
      <Outlet />
    </>
  );
}
