import { Link, useSearchParams } from "react-router";
import { z } from "zod";

import { TransactionList } from "@/app/transactions/_components/transaction-list";
import { TransactionsFilter } from "@/app/transactions/_components/transactions-filter";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { Button } from "@/components/ui/button";
import { useAllAccounts } from "@/lib/queries/accounts";
import { useTransactions } from "@/lib/queries/transactions";

const transactionsSearchSchema = z.object({
  accountId: z.uuid().optional(),
  hide: z.array(z.uuid()).default([]),
});

function hideSearchParams(hide: string[]): URLSearchParams {
  const params = new URLSearchParams();

  for (const accountId of hide) {
    params.append("hide", accountId);
  }

  return params;
}

function TransactionResults({
  accountId,
  excludeAccountId,
}: {
  accountId: string | undefined;
  excludeAccountId: string[];
}) {
  const transactions = useTransactions(accountId, excludeAccountId);
  const accounts = useAllAccounts();
  const accountsById = new Map(accounts.data?.map((account) => [account.id, account] as const));

  if (transactions.isPending) {
    return <LoadingState />;
  }

  if (transactions.isError) {
    return <ErrorState error={transactions.error} onRetry={transactions.refetch} />;
  }

  if (transactions.data.length === 0) {
    return <EmptyState title="No transactions yet" />;
  }

  return (
    <TransactionList
      transactions={transactions.data}
      accountsById={accountsById}
      hasMore={transactions.hasNextPage}
      loadingMore={transactions.isFetchingNextPage}
      onLoadMore={transactions.fetchNextPage}
    />
  );
}

export function TransactionsPage() {
  const [searchParams] = useSearchParams();
  const search = transactionsSearchSchema.safeParse({
    accountId: searchParams.get("accountId") ?? undefined,
    hide: searchParams.getAll("hide"),
  });

  if (!search.success) {
    return (
      <>
        <EmptyState title="Invalid account filter" />
        <Button variant="outline" asChild>
          <Link to="/transactions">All accounts</Link>
        </Button>
      </>
    );
  }

  const { accountId, hide } = search.data;

  return (
    <>
      <header className="flex items-center justify-between gap-2">
        <h1 className="text-xl font-semibold tracking-tight">Transactions</h1>
        <div className="flex items-center gap-2">
          {accountId === undefined ? <TransactionsFilter /> : null}
          {accountId === undefined ? null : (
            <Button variant="outline" size="sm" asChild>
              <Link to={{ pathname: "/transactions", search: hideSearchParams(hide).toString() }}>
                All accounts
              </Link>
            </Button>
          )}
        </div>
      </header>
      <TransactionResults accountId={accountId} excludeAccountId={hide} />
    </>
  );
}
