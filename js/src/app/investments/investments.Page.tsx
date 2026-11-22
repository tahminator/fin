import { Link } from "react-router";

import { InvestmentsView } from "@/app/investments/_components/investments-view";
import { useInvestmentsSearch } from "@/app/investments/_hooks/use-investments-search";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { useAllAccounts } from "@/lib/queries/accounts";

function InvestmentsContent() {
  const { search, update } = useInvestmentsSearch();
  const accounts = useAllAccounts();

  if (search === null) {
    return (
      <EmptyState
        title="Invalid chart settings"
        action={
          <Button variant="outline" asChild>
            <Link to="/investments">Reset</Link>
          </Button>
        }
      />
    );
  }

  if (accounts.isPending) {
    return <LoadingState />;
  }

  if (accounts.isError) {
    return <ErrorState error={accounts.error} onRetry={accounts.refetch} />;
  }

  const investmentAccounts = accounts.data.filter((account) => account.type === "investment");

  if (investmentAccounts.length === 0) {
    return (
      <EmptyState
        title="No investment accounts"
        description="Connect a brokerage or retirement account to track its value here."
        action={
          <Button asChild>
            <Link to="/connections">Go to connections</Link>
          </Button>
        }
      />
    );
  }

  return (
    <InvestmentsView
      accounts={investmentAccounts}
      range={search.range}
      off={search.off}
      update={update}
    />
  );
}

export function InvestmentsPage() {
  return (
    <>
      <PageHeader title="Investments" />
      <InvestmentsContent />
    </>
  );
}
