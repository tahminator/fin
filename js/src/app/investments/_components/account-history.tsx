import { historyLabel } from "@/app/investments/_components/portfolio-overview";
import { ErrorState } from "@/components/error-state";
import { InvestmentValueChart } from "@/components/investment-value-chart";
import { Skeleton } from "@/components/ui/skeleton";
import { useInvestmentHistory } from "@/lib/queries/investments";
import type { InvestmentHistory, InvestmentRange } from "@/lib/types/api";

export function AccountChart({
  accountId,
  range,
  securityIds,
}: {
  accountId: string;
  range: InvestmentRange;
  securityIds: string[] | null;
}) {
  if (securityIds === null) {
    return (
      <p className="type-footnote text-muted-foreground">No holdings selected for the chart.</p>
    );
  }

  return <AccountHistory accountId={accountId} range={range} securityIds={securityIds} />;
}

function AccountHistory({
  accountId,
  range,
  securityIds,
}: {
  accountId: string;
  range: InvestmentRange;
  securityIds: string[];
}) {
  const history = useInvestmentHistory(range, { excludeAccountId: [] }, securityIds);

  if (history.isPending) {
    return <Skeleton className="h-44 w-full" />;
  }

  if (history.isError) {
    return <ErrorState error={history.error} onRetry={history.refetch} />;
  }

  return (
    <AccountSeries history={history.data} accountId={accountId} label={historyLabel(securityIds)} />
  );
}

function AccountSeries({
  history,
  accountId,
  label,
}: {
  history: InvestmentHistory;
  accountId: string;
  label: string;
}) {
  const account = history.accounts.find((entry) => entry.accountId === accountId);

  if (account === undefined) {
    return (
      <p className="type-footnote text-muted-foreground">
        No value history for this account in this range yet. History will build as your accounts
        sync.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      {account.series.map((series) => (
        <InvestmentValueChart
          key={series.currency}
          series={series}
          label={label}
          className="h-36"
        />
      ))}
    </div>
  );
}
