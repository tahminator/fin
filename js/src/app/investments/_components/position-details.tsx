import { ListRow, ListSection } from "@/components/list-section";
import {
  formatMoney,
  formatPercent,
  formatShares,
  formatSignedMoney,
  isNegative,
} from "@/lib/format";
import type { PositionPerformance } from "@/lib/types/api";
import { cn } from "@/lib/utils";

function DetailRow({ label, value, tone }: { label: string; value: string; tone?: "up" | "down" }) {
  return (
    <ListRow className="justify-between">
      <span className="type-body text-muted-foreground">{label}</span>
      <span
        className={cn(
          "type-body tabular-nums",
          tone === "up" ? "text-success" : null,
          tone === "down" ? "text-destructive" : null,
        )}
      >
        {value}
      </span>
    </ListRow>
  );
}

export function PositionDetails({
  position,
  currency,
}: {
  position: PositionPerformance;
  currency: string;
}) {
  return (
    <ListSection
      header="Position"
      footer={
        position.marketPriced
          ? "Prices from Alpaca; charts trail the market by about 15 minutes."
          : "No market data for this holding; values use your brokerage's latest price."
      }
    >
      <DetailRow label="Shares" value={formatShares(position.quantity)} />
      <DetailRow label="Price" value={formatMoney(position.price, currency)} />
      <DetailRow label="Market value" value={formatMoney(position.value, currency)} />
      {position.averageCost === null ? null : (
        <DetailRow label="Average cost" value={formatMoney(position.averageCost, currency)} />
      )}
      {position.totalReturn === null ? null : (
        <DetailRow
          label="Total return"
          value={
            position.totalReturnPercent === null
              ? formatSignedMoney(position.totalReturn, currency)
              : `${formatSignedMoney(position.totalReturn, currency)} (${formatPercent(position.totalReturnPercent)})`
          }
          tone={isNegative(position.totalReturn) ? "down" : "up"}
        />
      )}
    </ListSection>
  );
}
