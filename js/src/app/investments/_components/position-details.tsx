import { ChevronRightIcon } from "lucide-react";
import { Link } from "react-router";

import { marketPerformanceLink } from "@/app/investments/_components/investments-search";
import { ListRow, ListSection } from "@/components/list-section";
import {
  formatMoney,
  formatPercent,
  formatShares,
  formatSignedMoney,
  isNegative,
} from "@/lib/format";
import type { PortfolioRange, PositionPerformance } from "@/lib/types/api";
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
  range,
  accountId,
}: {
  position: PositionPerformance;
  currency: string;
  range: PortfolioRange;
  accountId?: string | undefined;
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
      {position.marketPriceChange === null ? null : (
        <ListRow className="p-0">
          <Link
            to={marketPerformanceLink(range, position.securityId, accountId)}
            state={{ title: position.ticker ?? position.name ?? "Market performance" }}
            preventScrollReset
            className="flex min-h-13 w-full items-center justify-between gap-3 px-4 py-2 outline-none focus-visible:bg-accent active:bg-fill"
          >
            <span className="type-body text-muted-foreground">Market price change</span>
            <span
              className={cn(
                "flex items-center gap-1 type-body tabular-nums",
                isNegative(position.marketPriceChange) ? "text-destructive" : "text-success",
              )}
            >
              {position.marketPriceChangePercent === null
                ? formatSignedMoney(position.marketPriceChange, currency)
                : `${formatSignedMoney(position.marketPriceChange, currency)} (${formatPercent(position.marketPriceChangePercent)})`}
              <ChevronRightIcon aria-hidden="true" className="size-4 text-muted-foreground" />
            </span>
          </Link>
        </ListRow>
      )}
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
