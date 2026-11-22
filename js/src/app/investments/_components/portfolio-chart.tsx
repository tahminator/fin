import { Area, AreaChart, ReferenceLine, XAxis, YAxis } from "recharts";

import { isIntraday } from "@/app/investments/_components/investments-search";
import type { ChartConfig } from "@/components/ui/chart";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart";
import { formatDate, formatDateTime, formatMoney } from "@/lib/format";
import type { ChangeTone } from "@/lib/format";
import type { PortfolioPoint, PortfolioRange } from "@/lib/types/api";

interface PortfolioChartProps {
  points: PortfolioPoint[];
  range: PortfolioRange;
  currency: string;
  tone: ChangeTone;
}

const toneColors = {
  up: "var(--success)",
  down: "var(--destructive)",
  flat: "var(--chart-2)",
} as const satisfies Record<ChangeTone, string>;

function formatPointTime(time: string, range: PortfolioRange): string {
  return isIntraday(range) ? formatDateTime(time) : formatDate(time.slice(0, 10));
}

export function PortfolioChart({ points, range, currency, tone }: PortfolioChartProps) {
  const color = toneColors[tone];
  const chartConfig = { value: { label: "Value", color } } satisfies ChartConfig;
  const data = points.map((point) => ({ time: point.time, value: Number(point.value) }));
  const startValue = data[0]?.value;

  return (
    <ChartContainer config={chartConfig} className="aspect-auto h-52 w-full">
      <AreaChart data={data} margin={{ top: 8, bottom: 8, left: 0, right: 0 }}>
        <XAxis dataKey="time" hide />
        <YAxis
          hide
          domain={[
            (dataMin: number) => dataMin - Math.max(dataMin * 0.002, 1),
            (dataMax: number) => dataMax + Math.max(dataMax * 0.002, 1),
          ]}
        />
        {startValue === undefined ? null : (
          <ReferenceLine
            y={startValue}
            stroke="var(--muted-foreground)"
            strokeDasharray="2 4"
            strokeOpacity={0.6}
          />
        )}
        <ChartTooltip
          cursor={{ stroke: "var(--muted-foreground)", strokeWidth: 1 }}
          content={
            <ChartTooltipContent
              hideIndicator
              labelFormatter={(label) => formatPointTime(String(label), range)}
              formatter={(value) => formatMoney(Number(value).toFixed(2), currency)}
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
  );
}
