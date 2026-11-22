import { ErrorState } from "@/components/error-state";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDate, formatMoney, isNegative } from "@/lib/format";
import type { useInvestmentTransactions } from "@/lib/queries/investments";
import type { InvestmentTransaction } from "@/lib/types/api";
import { cn } from "@/lib/utils";

const zeroPattern = /^-?0(?:\.0+)?$/u;

function activityTitle(transaction: InvestmentTransaction): string {
  const security = [transaction.ticker, transaction.securityName]
    .filter((part) => part !== null && part !== "")
    .join(" · ");

  return security === "" ? transaction.name : security;
}

function ActivityRow({ transaction }: { transaction: InvestmentTransaction }) {
  const fees =
    transaction.fees === null || zeroPattern.test(transaction.fees)
      ? null
      : `Fees ${formatMoney(transaction.fees, transaction.currency)}`;
  const details = [
    formatDate(transaction.postedOn),
    transaction.type,
    transaction.accountName,
    fees,
  ]
    .filter((detail) => detail !== null)
    .join(" · ");

  return (
    <ListRow>
      <ListRowLabel
        title={activityTitle(transaction)}
        subtitle={<span className="first-letter:uppercase">{details}</span>}
      />
      <p
        className={cn(
          "shrink-0 type-body tabular-nums",
          isNegative(transaction.amount) || zeroPattern.test(transaction.amount)
            ? null
            : "text-success",
        )}
      >
        {formatMoney(transaction.amount, transaction.currency)}
      </p>
    </ListRow>
  );
}

export function InvestmentActivitySection({
  activity,
}: {
  activity: ReturnType<typeof useInvestmentTransactions>;
}) {
  if (activity.isPending) {
    return <Skeleton className="h-28 w-full rounded-[26px] bg-card" />;
  }

  if (activity.isError) {
    return <ErrorState error={activity.error} onRetry={activity.refetch} />;
  }

  if (activity.data.length === 0) {
    return null;
  }

  return (
    <ListSection header="Investment activity">
      {activity.data.map((transaction) => (
        <ActivityRow key={transaction.id} transaction={transaction} />
      ))}
      {activity.hasNextPage ? (
        <ListRow className="p-0">
          <LoadMoreButton
            loading={activity.isFetchingNextPage}
            onLoadMore={activity.fetchNextPage}
          />
        </ListRow>
      ) : null}
    </ListSection>
  );
}
