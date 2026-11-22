import { investmentRanges } from "@/app/investments/_hooks/use-investments-search";
import { Button } from "@/components/ui/button";
import type { InvestmentRange } from "@/lib/types/api";

const rangeLabels: Record<InvestmentRange, { short: string; long: string }> = {
  "1d": { short: "1D", long: "Last day" },
  "3d": { short: "3D", long: "Last 3 days" },
  "1w": { short: "1W", long: "Last week" },
  "1m": { short: "1M", long: "Last month" },
  "3m": { short: "3M", long: "Last 3 months" },
  "1y": { short: "1Y", long: "Last year" },
  all: { short: "ALL", long: "All history" },
};

export function rangeDescription(range: InvestmentRange): string {
  return rangeLabels[range].long;
}

export function RangeSelector({
  value,
  onChange,
}: {
  value: InvestmentRange;
  onChange: (range: InvestmentRange) => void;
}) {
  return (
    <fieldset className="flex gap-1 rounded-full bg-card p-1">
      <legend className="sr-only">Chart range</legend>
      {investmentRanges.map((range) => (
        <Button
          key={range}
          size="sm"
          variant={range === value ? "secondary" : "ghost"}
          className="flex-1 rounded-full px-0 tabular-nums"
          aria-pressed={range === value}
          aria-label={rangeLabels[range].long}
          onClick={() => {
            onChange(range);
          }}
        >
          {rangeLabels[range].short}
        </Button>
      ))}
    </fieldset>
  );
}
