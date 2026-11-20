import { Link, useSearchParams } from "react-router";
import { z } from "zod";

import { TransactionList } from "@/app/transactions/_components/transaction-list";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { Button } from "@/components/ui/button";
import { useTransactions } from "@/lib/queries/transactions";

const transactionsSearchSchema = z.object({
  accountId: z.uuid().optional(),
});

function TransactionResults({ accountId }: { accountId: string | undefined }) {
  const transactions = useTransactions(accountId);

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

  const { accountId } = search.data;

  return (
    <>
      <header className="flex items-center justify-between gap-2">
        <h1 className="text-xl font-semibold tracking-tight">Transactions</h1>
        {accountId === undefined ? null : (
          <Button variant="outline" size="sm" asChild>
            <Link to="/transactions">All accounts</Link>
          </Button>
        )}
      </header>
      <TransactionResults accountId={accountId} />
    </>
  );
}
