import { SecurityRow } from "@/app/investments/_components/security-row";
import { ErrorState } from "@/components/error-state";
import { ListSection } from "@/components/list-section";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { groupBy } from "@/lib/group-by";
import { useInvestmentHoldings, useInvestmentSecurities } from "@/lib/queries/investments";

interface SecurityControlsProps {
  off: string[];
  onToggle: (securityId: string, show: boolean) => void;
  onShowAll: () => void;
  onHideAll: (securityIds: string[]) => void;
}

export function SecurityControls({ off, onToggle, onShowAll, onHideAll }: SecurityControlsProps) {
  const securities = useInvestmentSecurities({ excludeAccountId: [] });
  const holdings = useInvestmentHoldings({ excludeAccountId: [] });

  if (securities.isPending || holdings.isPending) {
    return <Skeleton className="h-40 w-full rounded-[26px] bg-card" />;
  }

  if (securities.isError) {
    return <ErrorState error={securities.error} onRetry={securities.refetch} />;
  }

  if (holdings.isError) {
    return <ErrorState error={holdings.error} onRetry={holdings.refetch} />;
  }

  if (securities.data.length === 0) {
    return null;
  }

  const holdingsBySecurity = groupBy(holdings.data, (holding) => holding.securityId);

  return (
    <section className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between gap-2 px-4">
        <h2 className="type-footnote text-muted-foreground uppercase">Holdings in the chart</h2>
        <div className="flex gap-1">
          <Button variant="ghost" size="xs" disabled={off.length === 0} onClick={onShowAll}>
            Show all
          </Button>
          <Button
            variant="ghost"
            size="xs"
            disabled={off.length === securities.data.length}
            onClick={() => {
              onHideAll(securities.data.map((security) => security.id));
            }}
          >
            Hide all
          </Button>
        </div>
      </div>
      <ListSection
        label="Holdings in the chart"
        footer="All on charts whole account values, including cash. Turning holdings off charts only the ones left on."
      >
        {securities.data.map((security) => (
          <SecurityRow
            key={security.id}
            security={security}
            holdings={holdingsBySecurity.get(security.id) ?? []}
            checked={!off.includes(security.id)}
            onToggle={onToggle}
          />
        ))}
      </ListSection>
    </section>
  );
}
