import { Link } from "react-router";

import { toneClasses } from "@/app/investments/_components/change-label";
import type { InvestmentsSearch } from "@/app/investments/_components/investments-search";
import { investmentsLink } from "@/app/investments/_components/investments-search";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { changeTone, formatMoney, formatPercent, formatShares } from "@/lib/format";
import type { PositionPerformance } from "@/lib/types/api";
import { cn } from "@/lib/utils";

function isCash(position: PositionPerformance): boolean {
  return position.securityType === "cash";
}

function positionTitle(position: PositionPerformance): string {
  return isCash(position) ? "Cash" : (position.ticker ?? position.name ?? "Unknown security");
}

function positionSubtitle(position: PositionPerformance): string {
  if (isCash(position)) {
    return "Uninvested cash";
  }

  return position.ticker === null || position.name === null
    ? formatShares(position.quantity)
    : `${position.name} · ${formatShares(position.quantity)}`;
}

function TickerBadge({ position }: { position: PositionPerformance }) {
  const label = isCash(position) ? "$" : (position.ticker ?? position.name ?? "?").slice(0, 4);

  return (
    <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-fill type-caption2 font-bold">
      {label}
    </span>
  );
}

interface PositionListProps {
  positions: PositionPerformance[];
  currency: string;
  search: InvestmentsSearch;
}

export function PositionList({ positions, currency, search }: PositionListProps) {
  if (positions.length === 0) {
    return null;
  }

  return (
    <ListSection
      header="Holdings"
      footer="Change is your market gain over the selected range; buying or selling shares doesn't count as gain or loss."
    >
      {positions.map((position) => (
        <ListRow key={position.securityId} className="p-0">
          <Link
            to={investmentsLink({ range: search.range, securityId: position.securityId })}
            className="flex min-h-15 min-w-0 flex-1 items-center gap-3 px-4 py-2 outline-none focus-visible:bg-accent active:bg-fill"
          >
            <TickerBadge position={position} />
            <ListRowLabel title={positionTitle(position)} subtitle={positionSubtitle(position)} />
            <div className="shrink-0 text-right">
              <p className="type-body tabular-nums">{formatMoney(position.value, currency)}</p>
              <p
                className={cn(
                  "type-subheadline tabular-nums",
                  toneClasses[changeTone(position.change)],
                )}
              >
                {position.changePercent === null
                  ? formatMoney(position.change, currency)
                  : formatPercent(position.changePercent)}
              </p>
            </div>
          </Link>
        </ListRow>
      ))}
    </ListSection>
  );
}
