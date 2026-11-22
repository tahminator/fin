import { useState } from "react";

import { AccountList } from "@/app/_components/home/account-list";
import { BalanceSummary } from "@/app/_components/home/balance-summary";
import { CashFlowSection } from "@/app/_components/home/cash-flow-section";
import { InvestmentsOverview } from "@/app/_components/home/investments-overview";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { useSummary } from "@/lib/queries/summary";
import type { Account } from "@/lib/types/api";

interface AccountsOverviewProps {
  accounts: Account[];
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => Promise<unknown>;
}

function isInvestment(account: Account): boolean {
  return account.type === "investment";
}

export function AccountsOverview({
  accounts,
  hasMore,
  loadingMore,
  onLoadMore,
}: AccountsOverviewProps) {
  const [hiddenAccountIds, setHiddenAccountIds] = useState<string[]>([]);
  const summary = useSummary(hiddenAccountIds);

  const traditional = accounts.filter((account) => !isInvestment(account));
  const investments = accounts.filter((account) => isInvestment(account));
  const investmentIds = investments.map((account) => account.id);

  const toggleAccount = (accountId: string) => {
    setHiddenAccountIds((current) =>
      current.includes(accountId)
        ? current.filter((id) => id !== accountId)
        : [...current, accountId],
    );
  };

  if (summary.isPending) {
    return <LoadingState />;
  }

  if (summary.isError) {
    return <ErrorState error={summary.error} onRetry={summary.refetch} />;
  }

  return (
    <>
      <BalanceSummary summary={summary.data} />
      {traditional.length > 0 || hasMore ? (
        <section aria-labelledby="home-accounts" className="flex flex-col gap-3">
          <h2 id="home-accounts" className="px-1 type-title2 font-bold">
            Accounts
          </h2>
          <CashFlowSection
            excludeAccountId={[...new Set([...hiddenAccountIds, ...investmentIds])]}
          />
          <AccountList
            header="Bank and card accounts"
            footer="Hidden accounts are left out of cash flow and the totals above."
            accounts={traditional}
            hiddenAccountIds={hiddenAccountIds}
            onToggleAccount={toggleAccount}
            hasMore={hasMore}
            loadingMore={loadingMore}
            onLoadMore={onLoadMore}
          />
        </section>
      ) : null}
      {investments.length > 0 ? (
        <InvestmentsOverview
          accounts={investments}
          hiddenAccountIds={hiddenAccountIds}
          onToggleAccount={toggleAccount}
        />
      ) : null}
    </>
  );
}
