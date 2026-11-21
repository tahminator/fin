import { TransactionList } from "@/app/transactions/_components/transaction-list";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { ListRow, ListSection } from "@/components/list-section";
import { Skeleton } from "@/components/ui/skeleton";
import { useAllAccounts } from "@/lib/queries/accounts";
import { useTransactions } from "@/lib/queries/transactions";

export function TransactionListSkeleton() {
  return (
    <div className="flex flex-col gap-6" aria-label="Loading transactions">
      {[
        ["one", "two", "three"],
        ["four", "five"],
      ].map((keys, sectionIndex) => (
        <ListSection key={keys[0]} header={sectionIndex === 0 ? "Loading..." : undefined}>
          {keys.map((key) => (
            <ListRow key={key}>
              <Skeleton className="size-8 rounded-full" />
              <div className="flex min-w-0 flex-1 flex-col gap-2">
                <Skeleton className="h-4 w-2/5" />
                <Skeleton className="h-3 w-1/3" />
              </div>
              <Skeleton className="h-4 w-16" />
            </ListRow>
          ))}
        </ListSection>
      ))}
    </div>
  );
}

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
    return <TransactionListSkeleton />;
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
