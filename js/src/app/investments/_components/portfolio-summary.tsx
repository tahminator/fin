import { ChangeLabel } from "@/app/investments/_components/change-label";
import type { InvestmentsSearch } from "@/app/investments/_components/investments-search";
import { rangeDescription } from "@/app/investments/_components/investments-search";
import { PortfolioChart } from "@/app/investments/_components/portfolio-chart";
import { RangePicker } from "@/app/investments/_components/range-picker";
import { Card, CardContent } from "@/components/ui/card";
import { changeTone, formatMoney } from "@/lib/format";
import type { PortfolioPerformance } from "@/lib/types/api";

interface PortfolioSummaryProps {
  title: string;
  performance: PortfolioPerformance;
  search: InvestmentsSearch;
}

export function PortfolioSummary({ title, performance, search }: PortfolioSummaryProps) {
  return (
    <Card>
      <CardContent className="flex flex-col gap-4">
        <div>
          <p className="type-subheadline text-muted-foreground">{title}</p>
          <p className="truncate type-large-title font-bold tabular-nums">
            {formatMoney(performance.endValue, performance.currency)}
          </p>
          <ChangeLabel
            change={performance.change}
            changePercent={performance.changePercent}
            currency={performance.currency}
            suffix={rangeDescription(performance.range)}
            className="type-subheadline"
          />
        </div>
        <PortfolioChart
          points={performance.points}
          range={performance.range}
          currency={performance.currency}
          tone={changeTone(performance.change)}
        />
        <RangePicker search={search} />
      </CardContent>
    </Card>
  );
}
