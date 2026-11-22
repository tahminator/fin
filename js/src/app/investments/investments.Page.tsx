import { Outlet } from "react-router";

import { PortfolioView } from "@/app/investments/_components/portfolio-view";
import { usePortfolioRange } from "@/app/investments/_hooks/use-portfolio-range";
import { PageHeader } from "@/components/page-header";

export function InvestmentsPage() {
  const range = usePortfolioRange();

  return (
    <>
      <PageHeader title="Investments" />
      <PortfolioView range={range} />
      <Outlet />
    </>
  );
}
