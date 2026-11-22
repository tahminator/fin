import { ChevronRightIcon } from "lucide-react";
import { Link } from "react-router";

import { AccountList } from "@/app/_components/home/account-list";
import { ErrorState } from "@/components/error-state";
import { InvestmentValueChart, SeriesHeadline } from "@/components/investment-value-chart";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useInvestmentHistory } from "@/lib/queries/investments";
import type { Account } from "@/lib/types/api";

const homeRange = "1m";

function PortfolioValue({ excludeAccountId }: { excludeAccountId: string[] }) {
  const history = useInvestmentHistory(homeRange, { excludeAccountId });

  if (history.isPending) {
    return <Skeleton className="h-64 w-full rounded-[26px] bg-card" />;
  }

  if (history.isError) {
    return <ErrorState error={history.error} onRetry={history.refetch} />;
  }

  if (history.data.total.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Portfolio value</CardTitle>
        </CardHeader>
        <CardContent className="type-subheadline text-muted-foreground">
          {history.data.earliestCapturedAt === null
            ? "No portfolio history yet. It starts with the next successful investment sync."
            : "No portfolio history in the last month for the visible accounts."}
        </CardContent>
      </Card>
    );
  }

  return (
    <section className="flex flex-col gap-3">
      {history.data.total.map((series) => (
        <Card key={series.currency}>
          <CardContent className="flex flex-col gap-4">
            <SeriesHeadline series={series} label="Portfolio value · last month" />
            <InvestmentValueChart series={series} label="Portfolio value" className="h-36" />
          </CardContent>
        </Card>
      ))}
    </section>
  );
}

interface InvestmentsOverviewProps {
  accounts: Account[];
  hiddenAccountIds: string[];
  onToggleAccount: (accountId: string) => void;
}

export function InvestmentsOverview({
  accounts,
  hiddenAccountIds,
  onToggleAccount,
}: InvestmentsOverviewProps) {
  const allHidden = accounts.every((account) => hiddenAccountIds.includes(account.id));

  return (
    <section aria-labelledby="home-investments" className="flex flex-col gap-3">
      <div className="flex items-center justify-between px-1">
        <h2 id="home-investments" className="type-title2 font-bold">
          Investments
        </h2>
        <Button variant="ghost" size="sm" asChild>
          <Link to="/investments">
            Details
            <ChevronRightIcon />
          </Link>
        </Button>
      </div>
      {allHidden ? (
        <Card>
          <CardContent className="type-subheadline text-muted-foreground">
            All investment accounts are hidden.
          </CardContent>
        </Card>
      ) : (
        <PortfolioValue excludeAccountId={hiddenAccountIds} />
      )}
      <AccountList
        header="Investment accounts"
        footer="Hidden accounts are left out of the portfolio value and the totals above."
        accounts={accounts}
        hiddenAccountIds={hiddenAccountIds}
        onToggleAccount={onToggleAccount}
        hasMore={false}
        loadingMore={false}
        onLoadMore={async () => {}}
      />
    </section>
  );
}
