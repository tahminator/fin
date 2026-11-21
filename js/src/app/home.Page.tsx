import { Link } from "react-router";

import { AccountsOverview } from "@/app/_components/home/accounts-overview";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { useAccounts } from "@/lib/queries/accounts";

function HomeContent() {
  const accounts = useAccounts();

  if (accounts.isPending) {
    return <LoadingState />;
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
    <AccountsOverview
      accounts={accounts.data}
      hasMore={accounts.hasNextPage}
      loadingMore={accounts.isFetchingNextPage}
      onLoadMore={accounts.fetchNextPage}
    />
  );
}

export function HomePage() {
  return (
    <>
      <PageHeader title="Home" />
      <HomeContent />
    </>
  );
}
