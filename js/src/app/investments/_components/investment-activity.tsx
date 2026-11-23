import { Link } from "react-router";

import { ErrorState } from "@/components/error-state";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDate, formatMoney, isNegative } from "@/lib/format";
import type { InvestmentScope } from "@/lib/queries/investments";
import { useInvestmentTransactions } from "@/lib/queries/investments";
import { cn } from "@/lib/utils";

export function InvestmentActivity({ scope }: { scope: InvestmentScope }) {
  const transactions = useInvestmentTransactions(scope);

  if (transactions.isPending) {
    return <Skeleton className="h-40 w-full rounded-[26px] bg-card" />;
  }

  if (transactions.isError) {
    return <ErrorState error={transactions.error} onRetry={transactions.refetch} />;
  }

  if (transactions.data.length === 0) {
    return null;
  }

  return (
    <ListSection header="Activity">
      {transactions.data.map((transaction) => (
        <ListRow key={transaction.id} className="p-0">
          <Link
            to={`/investments/transactions/${transaction.id}`}
            state={{ title: transaction.name }}
            preventScrollReset
            className="flex min-h-13 w-full items-center gap-3 px-4 py-2 active:bg-fill"
          >
            <ListRowLabel
              title={
                transaction.ticker !== null &&
                transaction.ticker !== "" &&
                !transaction.ticker.startsWith("CUR:")
                  ? `${transaction.ticker} · ${transaction.name}`
                  : transaction.name
              }
              subtitle={
                <span className="capitalize">
                  {`${formatDate(transaction.postedOn)} · ${transaction.subtype} · ${transaction.accountName}`}
                </span>
              }
            />
            <p
              className={cn(
                "shrink-0 type-body tabular-nums",
                isNegative(transaction.amount) ? null : "text-success",
              )}
            >
              {formatMoney(transaction.amount, transaction.currency)}
            </p>
          </Link>
        </ListRow>
      ))}
      {transactions.hasNextPage ? (
        <ListRow className="p-0">
          <LoadMoreButton
            loading={transactions.isFetchingNextPage}
            onLoadMore={transactions.fetchNextPage}
          />
        </ListRow>
      ) : null}
    </ListSection>
  );
}
