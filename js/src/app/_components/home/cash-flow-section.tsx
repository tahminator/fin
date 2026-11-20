import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";

import { ErrorState } from "@/components/error-state";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { ChartConfig } from "@/components/ui/chart";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDate, formatMoney, formatShortDate } from "@/lib/format";
import { useCashFlow } from "@/lib/queries/summary";
import type { CurrencyCashFlow } from "@/lib/types/api";

const chartConfig = {
  in: {
    label: "In",
    theme: { light: "var(--color-emerald-600)", dark: "var(--color-emerald-400)" },
  },
  out: {
    label: "Out",
    color: "var(--chart-2)",
  },
} satisfies ChartConfig;

function total(amounts: string[]): string {
  return amounts.reduce((sum, amount) => sum + Number(amount), 0).toFixed(2);
}

const compactFormatter = new Intl.NumberFormat(undefined, {
  notation: "compact",
  maximumFractionDigits: 1,
});

function CurrencyCashFlowCard({ flow }: { flow: CurrencyCashFlow }) {
  const data = flow.days.map((day) => ({
    date: day.date,
    in: Number(day.in),
    out: Number(day.out),
  }));

  const totalIn = total(flow.days.map((day) => day.in));
  const totalOut = total(flow.days.map((day) => day.out));

  return (
    <Card>
      <CardHeader>
        <CardDescription>
          Cash flow · last {flow.days.length} days · {flow.currency}
        </CardDescription>
        <CardTitle className="flex flex-wrap gap-x-6 gap-y-1 tabular-nums">
          <span className="text-emerald-600 dark:text-emerald-400">
            In {formatMoney(totalIn, flow.currency)}
          </span>
          <span>Out {formatMoney(totalOut, flow.currency)}</span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <ChartContainer config={chartConfig} className="aspect-auto h-48 w-full">
          <AreaChart data={data} margin={{ top: 12, left: 0, right: 8 }}>
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="date"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              minTickGap={24}
              tickFormatter={formatShortDate}
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              width={48}
              domain={[0, (dataMax: number) => Math.ceil(dataMax * 1.1)]}
              tickFormatter={(value: number) => compactFormatter.format(value)}
            />
            <ChartTooltip
              content={
                <ChartTooltipContent labelFormatter={(label) => formatDate(String(label))} />
              }
            />
            <Area
              dataKey="in"
              type="monotone"
              stroke="var(--color-in)"
              fill="var(--color-in)"
              fillOpacity={0.15}
              strokeWidth={2}
              dot={false}
            />
            <Area
              dataKey="out"
              type="monotone"
              stroke="var(--color-out)"
              fill="var(--color-out)"
              fillOpacity={0.15}
              strokeWidth={2}
              dot={false}
            />
          </AreaChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

export function CashFlowSection({ excludeAccountId }: { excludeAccountId: string[] }) {
  const cashFlow = useCashFlow(excludeAccountId);

  if (cashFlow.isPending) {
    return <Skeleton className="h-72 w-full rounded-xl" />;
  }

  if (cashFlow.isError) {
    return <ErrorState error={cashFlow.error} onRetry={cashFlow.refetch} />;
  }

  if (cashFlow.data.currencies.length === 0) {
    return null;
  }

  return (
    <section className="flex flex-col gap-3">
      {cashFlow.data.currencies.map((flow) => (
        <CurrencyCashFlowCard key={flow.currency} flow={flow} />
      ))}
    </section>
  );
}
