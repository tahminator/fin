import { ClockIcon } from "lucide-react";

import { InstitutionLogo } from "@/components/institution-logo";
import { LoadMoreButton } from "@/components/load-more-button";
import { formatDate, formatMoney, isNegative } from "@/lib/format";
import type { Account, Transaction } from "@/lib/types/api";
import { cn } from "@/lib/utils";

function TransactionRow({
  transaction,
  accountsById,
}: {
  transaction: Transaction;
  accountsById: ReadonlyMap<string, Account>;
}) {
  const account = accountsById.get(transaction.accountId);

  return (
    <li className="flex items-center justify-between gap-4 py-3">
      <div className="flex min-w-0 items-center gap-3">
        <InstitutionLogo url={account?.institutionLogoUrl ?? null} />
        <div className="min-w-0">
          <p className="flex items-center gap-2 font-medium">
            <span className="truncate">{transaction.merchantName ?? transaction.name}</span>
            {transaction.pending ? (
              <ClockIcon className="size-4 shrink-0 text-muted-foreground" aria-label="Pending" />
            ) : null}
          </p>
          <p className="truncate text-sm text-muted-foreground">
            {formatDate(transaction.postedOn)} · {transaction.accountName}
          </p>
        </div>
      </div>
      <p
        className={cn(
          "shrink-0 font-medium tabular-nums",
          !isNegative(transaction.amount) && "text-positive",
        )}
      >
        {formatMoney(transaction.amount, transaction.currency)}
      </p>
    </li>
  );
}

interface TransactionListProps {
  transactions: Transaction[];
  accountsById: ReadonlyMap<string, Account>;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => Promise<unknown>;
}

export function TransactionList({
  transactions,
  accountsById,
  hasMore,
  loadingMore,
  onLoadMore,
}: TransactionListProps) {
  return (
    <section className="flex flex-col gap-2">
      <ul className="divide-y">
        {transactions.map((transaction) => (
          <TransactionRow
            key={transaction.id}
            transaction={transaction}
            accountsById={accountsById}
          />
        ))}
      </ul>
      {hasMore ? <LoadMoreButton loading={loadingMore} onLoadMore={onLoadMore} /> : null}
    </section>
  );
}
