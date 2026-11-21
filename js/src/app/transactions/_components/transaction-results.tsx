import { TransactionList } from "@/app/transactions/_components/transaction-list";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { useAllAccounts } from "@/lib/queries/accounts";
import { useTransactions } from "@/lib/queries/transactions";

export function TransactionResults({
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
