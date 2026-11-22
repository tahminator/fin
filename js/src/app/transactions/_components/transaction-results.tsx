import { TransactionList } from "@/app/transactions/_components/transaction-list";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { useAllAccounts } from "@/lib/queries/accounts";
import type { useTransactions } from "@/lib/queries/transactions";

export function TransactionResults({
  transactions,
  showHeading,
}: {
  transactions: ReturnType<typeof useTransactions>;
  showHeading: boolean;
}) {
  const accounts = useAllAccounts();
  const accountsById = new Map(accounts.data?.map((account) => [account.id, account] as const));

  if (transactions.isPending) {
    return <LoadingState />;
  }

  if (transactions.isError) {
    return <ErrorState error={transactions.error} onRetry={transactions.refetch} />;
  }

  if (transactions.data.length === 0) {
    return showHeading ? (
      <p className="px-4 type-footnote text-muted-foreground">
        No bank or card transactions in this view.
      </p>
    ) : null;
  }

  return (
    <section className="flex flex-col gap-3">
      {showHeading ? (
        <h2 className="px-4 type-footnote text-muted-foreground uppercase">
          Bank and card transactions
        </h2>
      ) : null}
      <TransactionList
        transactions={transactions.data}
        accountsById={accountsById}
        hasMore={transactions.hasNextPage}
        loadingMore={transactions.isFetchingNextPage}
        onLoadMore={transactions.fetchNextPage}
      />
    </section>
  );
}
