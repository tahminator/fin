import { rangeDescription } from "@/app/investments/_components/range-selector";
import { ErrorState } from "@/components/error-state";
import { InvestmentValueChart, SeriesHeadline } from "@/components/investment-value-chart";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDateTime } from "@/lib/format";
import { useInvestmentHistory } from "@/lib/queries/investments";
import type { InvestmentRange } from "@/lib/types/api";

export function historyLabel(securityIds: string[]): string {
  return securityIds.length === 0 ? "Portfolio value" : "Selected holdings value";
}

function Notice({ children }: { children: string }) {
  return (
    <Card>
      <CardContent className="type-subheadline text-muted-foreground">{children}</CardContent>
    </Card>
  );
}

export function PortfolioOverview({
  range,
  securityIds,
}: {
  range: InvestmentRange;
  securityIds: string[] | null;
}) {
  if (securityIds === null) {
    return <Notice>No holdings selected. Turn on a holding below to chart its value.</Notice>;
  }

  return <PortfolioHistory range={range} securityIds={securityIds} />;
}

function PortfolioHistory({
  range,
  securityIds,
}: {
  range: InvestmentRange;
  securityIds: string[];
}) {
  const history = useInvestmentHistory(range, { excludeAccountId: [] }, securityIds);
  const label = historyLabel(securityIds);

  if (history.isPending) {
    return <Skeleton className="h-72 w-full rounded-[26px] bg-card" />;
  }

  if (history.isError) {
    return <ErrorState error={history.error} onRetry={history.refetch} />;
  }

  if (history.data.earliestCapturedAt === null) {
    return (
      <Notice>
        No portfolio history yet. History will build as your accounts sync, starting with the next
        successful investment sync.
      </Notice>
    );
  }

  if (history.data.total.length === 0) {
    return <Notice>{`No ${label.toLowerCase()} was captured in this range.`}</Notice>;
  }

  return (
    <section aria-label={label} className="flex flex-col gap-3">
      {history.data.total.map((series) => (
        <Card key={series.currency}>
          <CardContent className="flex flex-col gap-4">
            <SeriesHeadline series={series} label={`${label} · ${rangeDescription(range)}`} />
            <InvestmentValueChart series={series} label={label} />
            {securityIds.length > 0 ? (
              <p className="type-footnote text-muted-foreground">
                Only the selected holdings are counted. Cash and other holdings are left out, so
                this can differ from account balances.
              </p>
            ) : null}
          </CardContent>
        </Card>
      ))}
      {history.data.latestCapturedAt === null ? null : (
        <p className="px-4 type-footnote text-muted-foreground">
          {`Values come from your account syncs, last captured ${formatDateTime(history.data.latestCapturedAt)}. Deposits and withdrawals change portfolio value too, so this is not investment return.`}
        </p>
      )}
    </section>
  );
}
