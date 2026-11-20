import { Link } from "react-router";

import { AccountList } from "@/app/_components/home/account-list";
import { BalanceSummary } from "@/app/_components/home/balance-summary";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { Button } from "@/components/ui/button";
import { useAccounts } from "@/lib/queries/accounts";
import { useSummary } from "@/lib/queries/summary";

export function HomePage() {
  const summary = useSummary();
  const accounts = useAccounts();

  if (summary.isPending || accounts.isPending) {
    return <LoadingState />;
  }

  if (summary.isError) {
    return <ErrorState error={summary.error} onRetry={summary.refetch} />;
  }

  if (accounts.isError) {
    return <ErrorState error={accounts.error} onRetry={accounts.refetch} />;
  }

  if (accounts.data.length === 0) {
    return (
      <EmptyState
        title="No accounts yet"
        description="Connect a bank to see your balances here."
        action={
          <Button asChild>
            <Link to="/connections">Connect a bank</Link>
          </Button>
        }
      />
    );
  }

  return (
    <>
      <BalanceSummary summary={summary.data} />
      <AccountList
        accounts={accounts.data}
        hasMore={accounts.hasNextPage}
        loadingMore={accounts.isFetchingNextPage}
        onLoadMore={accounts.fetchNextPage}
      />
    </>
  );
}
