import { ErrorState } from "@/components/error-state";
import { ListRow, ListRowLabel } from "@/components/list-section";
import { Skeleton } from "@/components/ui/skeleton";
import { formatMoney } from "@/lib/format";
import { useInvestmentHoldings } from "@/lib/queries/investments";
import { securityLabel } from "@/lib/securities";

export function AccountHoldings({ accountId }: { accountId: string }) {
  const holdings = useInvestmentHoldings({ excludeAccountId: [] });

  if (holdings.isPending) {
    return <Skeleton className="h-16 w-full" />;
  }

  if (holdings.isError) {
    return <ErrorState error={holdings.error} onRetry={holdings.refetch} />;
  }

  const owned = holdings.data.filter((holding) => holding.accountId === accountId);

  if (owned.length === 0) {
    return <p className="type-footnote text-muted-foreground">No current holdings.</p>;
  }

  return (
    <ul aria-label="Current holdings" className="-mx-4">
      {owned.map((holding) => (
        <ListRow key={holding.id} className="not-first:before:left-4">
          <ListRowLabel
            title={securityLabel({
              id: holding.securityId,
              ticker: holding.ticker,
              name: holding.securityName,
            })}
            subtitle={`${Number(holding.quantity).toLocaleString()} units at ${formatMoney(holding.institutionPrice, holding.currency)}`}
          />
          <p className="shrink-0 type-body tabular-nums">
            {formatMoney(holding.institutionValue, holding.currency)}
          </p>
        </ListRow>
      ))}
    </ul>
  );
}
