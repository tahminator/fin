import { ErrorState } from "@/components/error-state";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { Skeleton } from "@/components/ui/skeleton";
import { formatMoney } from "@/lib/format";
import type { useInvestmentHoldings } from "@/lib/queries/investments";
import { securityLabel } from "@/lib/securities";
import type { InvestmentHolding } from "@/lib/types/api";

function HoldingRow({ holding }: { holding: InvestmentHolding }) {
  const details = [
    `${Number(holding.quantity).toLocaleString()} units`,
    holding.securityType,
    holding.accountName,
  ]
    .filter((detail) => detail !== null)
    .join(" · ");

  return (
    <ListRow>
      <ListRowLabel
        title={securityLabel({
          id: holding.securityId,
          ticker: holding.ticker,
          name: holding.securityName,
        })}
        subtitle={<span className="first-letter:uppercase">{details}</span>}
      />
      <p className="shrink-0 type-body tabular-nums">
        {formatMoney(holding.institutionValue, holding.currency)}
      </p>
    </ListRow>
  );
}

export function InvestmentHoldingsSection({
  holdings,
}: {
  holdings: ReturnType<typeof useInvestmentHoldings>;
}) {
  if (holdings.isPending) {
    return <Skeleton className="h-28 w-full rounded-[26px] bg-card" />;
  }

  if (holdings.isError) {
    return <ErrorState error={holdings.error} onRetry={holdings.refetch} />;
  }

  if (holdings.data.length === 0) {
    return null;
  }

  return (
    <ListSection header="Holdings">
      {holdings.data.map((holding) => (
        <HoldingRow key={holding.id} holding={holding} />
      ))}
    </ListSection>
  );
}
