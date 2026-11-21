import { Link } from "react-router";

import { AccountsOverview } from "@/app/_components/home/accounts-overview";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useAccounts } from "@/lib/queries/accounts";

function HomeSkeleton() {
  return (
    <div className="flex flex-col gap-6" aria-label="Loading home">
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-4 rounded-[26px] border p-4">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-10 w-48" />
          <div className="grid grid-cols-2 gap-3">
            <Skeleton className="h-16 rounded-2xl" />
            <Skeleton className="h-16 rounded-2xl" />
          </div>
        </div>
      </div>
      <Skeleton className="h-80 w-full rounded-[26px]" />
      <div className="flex flex-col gap-1.5 rounded-[26px] border p-2">
        {["one", "two", "three"].map((key) => (
          <div key={key} className="flex min-h-15 items-center gap-3 px-2">
            <Skeleton className="size-9 rounded-full" />
            <div className="flex flex-1 flex-col gap-2">
              <Skeleton className="h-4 w-2/5" />
              <Skeleton className="h-3 w-1/3" />
            </div>
            <Skeleton className="h-4 w-16" />
          </div>
        ))}
      </div>
    </div>
  );
}

function HomeContent() {
  const accounts = useAccounts();

  if (accounts.isPending) {
    return <HomeSkeleton />;
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
