import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";

import type { ChartConfig } from "@/components/ui/chart";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart";
import { formatDateTime, formatMoney, isNegative } from "@/lib/format";
import type { InvestmentSeries } from "@/lib/types/api";
import { cn } from "@/lib/utils";

const compactFormatter = new Intl.NumberFormat(undefined, {
  notation: "compact",
  maximumFractionDigits: 1,
});

const tickFormatter = new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" });

function tooltipLabel(_label: unknown, payload: readonly { payload?: unknown }[]): string {
  const point = payload[0]?.payload;

  if (
    typeof point !== "object" ||
    point === null ||
    !("at" in point) ||
    typeof point.at !== "number"
  ) {
    return "";
  }

  return formatDateTime(new Date(point.at).toISOString());
}

const limitedHistoryMessage = "History will build as your accounts sync.";

function currencyOf(series: InvestmentSeries): string | null {
  return series.currency === "" ? null : series.currency;
}

export function SeriesHeadline({ series, label }: { series: InvestmentSeries; label: string }) {
  const currency = currencyOf(series);
  const hasChange = series.points.length > 1;
  const down = isNegative(series.change);

  return (
    <div className="flex flex-col gap-0.5">
      <p className="type-subheadline text-muted-foreground">
        {label}
        {currency === null ? null : ` · ${currency}`}
      </p>
      <p className="truncate type-title1 font-bold tabular-nums">
        {formatMoney(series.current, currency)}
      </p>
      {hasChange ? (
        <p className="type-subheadline tabular-nums">
          <span className={cn("font-medium", down ? "text-destructive" : "text-success")}>
            {down ? "" : "+"}
            {formatMoney(series.change, currency)}
            {series.changePercent === null ? null : ` (${down ? "" : "+"}${series.changePercent}%)`}
          </span>
          <span className="text-muted-foreground"> in this range</span>
        </p>
      ) : null}
    </div>
  );
}

interface InvestmentValueChartProps {
  series: InvestmentSeries;
  label: string;
  className?: string;
}

export function InvestmentValueChart({ series, label, className }: InvestmentValueChartProps) {
  const currency = currencyOf(series);
  const color = isNegative(series.change) ? "var(--destructive)" : "var(--success)";

  const chartConfig = {
    value: { label, color },
  } satisfies ChartConfig;

  if (series.points.length < 2) {
    return <p className="type-footnote text-muted-foreground">{limitedHistoryMessage}</p>;
  }

  const data = series.points.map((point) => ({
    at: new Date(point.at).getTime(),
    value: Number(point.value),
  }));

  return (
    <div className="flex flex-col gap-2">
      <ChartContainer
        config={chartConfig}
        className={cn("aspect-auto h-44 w-full", className)}
        aria-label={`${label} chart`}
      >
        <AreaChart data={data} margin={{ top: 12, left: 0, right: 12 }}>
          <CartesianGrid vertical={false} />
          <XAxis
            dataKey="at"
            type="number"
            scale="time"
            domain={["dataMin", "dataMax"]}
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            minTickGap={32}
            tickFormatter={(value: number) => tickFormatter.format(value)}
          />
          <YAxis
            tickLine={false}
            axisLine={false}
            width={52}
            domain={["auto", "auto"]}
            tickFormatter={(value: number) => compactFormatter.format(value)}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                labelFormatter={tooltipLabel}
                formatter={(value) => `${label}: ${formatMoney(String(value), currency)}`}
              />
            }
          />
          <Area
            dataKey="value"
            type="linear"
            stroke="var(--color-value)"
            fill="var(--color-value)"
            fillOpacity={0.12}
            strokeWidth={2}
            dot={false}
            isAnimationActive={false}
          />
        </AreaChart>
      </ChartContainer>
      {series.limited ? (
        <p className="type-footnote text-muted-foreground">{limitedHistoryMessage}</p>
      ) : null}
    </div>
  );
}
