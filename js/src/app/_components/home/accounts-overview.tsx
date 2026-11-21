import { useState } from "react";

import { AccountList } from "@/app/_components/home/account-list";
import { BalanceSummary } from "@/app/_components/home/balance-summary";
import { CashFlowSection } from "@/app/_components/home/cash-flow-section";
import { ErrorState } from "@/components/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { useSummary } from "@/lib/queries/summary";
import type { Account } from "@/lib/types/api";

interface AccountsOverviewProps {
  accounts: Account[];
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => Promise<unknown>;
}

function AccountsOverviewSkeleton() {
  return (
    <div className="flex flex-col gap-6" aria-label="Loading account overview">
      <div className="flex flex-col gap-4 rounded-[26px] border p-4">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-10 w-52" />
        <div className="grid grid-cols-2 gap-3">
          <Skeleton className="h-16 rounded-2xl" />
          <Skeleton className="h-16 rounded-2xl" />
        </div>
      </div>
      <Skeleton className="h-80 w-full rounded-[26px]" />
    </div>
  );
}

export function AccountsOverview({
  accounts,
  hasMore,
  loadingMore,
  onLoadMore,
}: AccountsOverviewProps) {
  const [hiddenAccountIds, setHiddenAccountIds] = useState<string[]>([]);
  const summary = useSummary(hiddenAccountIds);

  if (summary.isPending) {
    return <AccountsOverviewSkeleton />;
  }

  if (summary.isError) {
    return <ErrorState error={summary.error} onRetry={summary.refetch} />;
  }

  return (
    <>
      <BalanceSummary summary={summary.data} />
      <CashFlowSection excludeAccountId={hiddenAccountIds} />
      <AccountList
        accounts={accounts}
        hiddenAccountIds={hiddenAccountIds}
        onToggleAccount={(accountId) => {
          setHiddenAccountIds((current) =>
            current.includes(accountId)
              ? current.filter((id) => id !== accountId)
              : [...current, accountId],
          );
        }}
        hasMore={hasMore}
        loadingMore={loadingMore}
        onLoadMore={onLoadMore}
      />
    </>
  );
}
