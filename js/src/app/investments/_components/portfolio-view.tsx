import { Link } from "react-router";

import type { InvestmentsSearch } from "@/app/investments/_components/investments-search";
import { PortfolioContent } from "@/app/investments/_components/portfolio-content";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { Button } from "@/components/ui/button";
import { usePortfolioPerformance } from "@/lib/queries/investments";

export function PortfolioView({ search }: { search: InvestmentsSearch }) {
  const performance = usePortfolioPerformance(search.range, {
    accountId: search.accountId,
    securityId: search.securityId,
  });

  if (performance.isPending) {
    return <LoadingState />;
  }

  if (performance.isError) {
    return <ErrorState error={performance.error} onRetry={performance.refetch} />;
  }

  const { data } = performance;
  const unscoped = search.accountId === undefined && search.securityId === undefined;

  if (data.positions.length === 0 && unscoped) {
    return (
      <EmptyState
        title="No investments yet"
        description="Connect a brokerage or retirement account to track your portfolio here."
        action={
          <Button asChild>
            <Link to="/connections">Connect an account</Link>
          </Button>
        }
      />
    );
  }

  if (data.positions.length === 0) {
    return <EmptyState title="Nothing held here" description="Try a longer time range." />;
  }

  return <PortfolioContent performance={data} search={search} />;
}
