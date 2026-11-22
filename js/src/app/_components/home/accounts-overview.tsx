import { useState } from "react";

import { AccountList } from "@/app/_components/home/account-list";
import { BalanceSummary } from "@/app/_components/home/balance-summary";
import { CashFlowSection } from "@/app/_components/home/cash-flow-section";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { useCashSummary } from "@/lib/queries/summary";
import type { Account } from "@/lib/types/api";

interface AccountsOverviewProps {
  accounts: Account[];
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => Promise<unknown>;
}

export function AccountsOverview({
  accounts,
  hasMore,
  loadingMore,
  onLoadMore,
}: AccountsOverviewProps) {
  const [hiddenAccountIds, setHiddenAccountIds] = useState<string[]>([]);
  const summary = useCashSummary(hiddenAccountIds);

  if (summary.isPending) {
    return <LoadingState />;
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
