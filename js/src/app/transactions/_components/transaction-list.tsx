import { LoadMoreButton } from "@/components/load-more-button";
import { Badge } from "@/components/ui/badge";
import { formatDate, formatMoney, isNegative } from "@/lib/format";
import type { Transaction } from "@/lib/types/api";
import { cn } from "@/lib/utils";

function TransactionRow({ transaction }: { transaction: Transaction }) {
  return (
    <li className="flex items-center justify-between gap-4 py-3">
      <div className="min-w-0">
        <p className="flex items-center gap-2 font-medium">
          <span className="truncate">{transaction.merchantName ?? transaction.name}</span>
          {transaction.pending ? <Badge variant="outline">Pending</Badge> : null}
        </p>
        <p className="truncate text-sm text-muted-foreground">
          {formatDate(transaction.postedOn)} · {transaction.accountName}
        </p>
      </div>
      <p
        className={cn(
          "shrink-0 font-medium tabular-nums",
          !isNegative(transaction.amount) && "text-emerald-600 dark:text-emerald-400",
        )}
      >
        {formatMoney(transaction.amount, transaction.currency)}
      </p>
    </li>
  );
}

interface TransactionListProps {
  transactions: Transaction[];
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => Promise<unknown>;
}

export function TransactionList({
  transactions,
  hasMore,
  loadingMore,
  onLoadMore,
}: TransactionListProps) {
  return (
    <section className="flex flex-col gap-2">
      <ul className="divide-y">
        {transactions.map((transaction) => (
          <TransactionRow key={transaction.id} transaction={transaction} />
        ))}
      </ul>
      {hasMore ? <LoadMoreButton loading={loadingMore} onLoadMore={onLoadMore} /> : null}
    </section>
  );
}
