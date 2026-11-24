import { ChangeLabel } from "@/app/investments/_components/change-label";
import { rangeDescription } from "@/app/investments/_components/investments-search";
import { PortfolioChart } from "@/app/investments/_components/portfolio-chart";
import { RangePicker } from "@/app/investments/_components/range-picker";
import { Card, CardContent } from "@/components/ui/card";
import { changeTone, formatMoney } from "@/lib/format";
import type { PortfolioPerformance, PortfolioPoint, PortfolioRange } from "@/lib/types/api";

interface PortfolioSummaryProps {
  title: string;
  performance: PortfolioPerformance;
}

export function PortfolioSummary({ title, performance }: PortfolioSummaryProps) {
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
        <RangePicker range={performance.range} />
      </CardContent>
    </Card>
  );
}

export function MarketPerformanceSummary({
  points,
  range,
  currency,
  change,
  changePercent,
}: {
  points: PortfolioPoint[];
  range: PortfolioRange;
  currency: string;
  change: string;
  changePercent: string | null;
}) {
  const latest = points.at(-1);

  if (latest === undefined) {
    return null;
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-4">
        <div>
          <p className="type-subheadline text-muted-foreground">Market price per share</p>
          <p className="truncate type-large-title font-bold tabular-nums">
            {formatMoney(latest.value, currency)}
          </p>
          <ChangeLabel
            change={change}
            changePercent={changePercent}
            currency={currency}
            suffix={rangeDescription(range)}
            className="type-subheadline"
          />
        </div>
        <PortfolioChart
          points={points}
          range={range}
          currency={currency}
          tone={changeTone(change)}
        />
        <RangePicker range={range} />
        <p className="px-1 type-footnote text-muted-foreground">
          Per-share market performance, independent of how many shares you own.
        </p>
      </CardContent>
    </Card>
  );
}
