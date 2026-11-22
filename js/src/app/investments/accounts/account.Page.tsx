import { Outlet, useParams } from "react-router";

import { AccountView } from "@/app/investments/_components/account-view";
import { DetailPanel } from "@/app/investments/_components/detail-panel";
import { portfolioLink } from "@/app/investments/_components/investments-search";
import { useDetailTitle } from "@/app/investments/_hooks/use-detail-title";
import { usePortfolioRange } from "@/app/investments/_hooks/use-portfolio-range";

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
