import { InstitutionLogo } from "@/components/institution-logo";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import { formatDate, formatMoney, isNegative } from "@/lib/format";
import type { Account, Transaction } from "@/lib/types/api";
import { cn } from "@/lib/utils";

interface DayGroup {
  date: string;
  transactions: Transaction[];
}

function groupByDay(transactions: Transaction[]): DayGroup[] {
  const groups: DayGroup[] = [];

  for (const transaction of transactions) {
    const current = groups.at(-1);

    if (current?.date === transaction.postedOn) {
      current.transactions.push(transaction);
    } else {
      groups.push({ date: transaction.postedOn, transactions: [transaction] });
    }
  }

  return groups;
}

function TransactionRow({
  transaction,
  accountsById,
}: {
  transaction: Transaction;
  accountsById: ReadonlyMap<string, Account>;
}) {
  const account = accountsById.get(transaction.accountId);

  return (
    <ListRow>
      <InstitutionLogo url={account?.institutionLogoUrl ?? null} />
      <ListRowLabel
        title={transaction.merchantName ?? transaction.name}
        subtitle={
          transaction.pending ? `Pending · ${transaction.accountName}` : transaction.accountName
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
    </ListRow>
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
    <div className="flex flex-col gap-6">
      {groupByDay(transactions).map((group) => (
        <ListSection key={group.date} header={formatDate(group.date)}>
          {group.transactions.map((transaction) => (
            <TransactionRow
              key={transaction.id}
              transaction={transaction}
              accountsById={accountsById}
            />
          ))}
        </ListSection>
      ))}
      {hasMore ? (
        <ListSection>
          <ListRow className="p-0">
            <LoadMoreButton loading={loadingMore} onLoadMore={onLoadMore} />
          </ListRow>
        </ListSection>
      ) : null}
    </div>
  );
}
