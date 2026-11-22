import { ArrowDownRightIcon, ArrowRightIcon, ArrowUpRightIcon } from "lucide-react";

import { changeTone, formatPercent, formatSignedMoney } from "@/lib/format";
import type { ChangeTone } from "@/lib/format";
import { cn } from "@/lib/utils";

interface ChangeLabelProps {
  change: string;
  changePercent: string | null;
  currency: string;
  suffix?: string;
  className?: string;
}

const toneIcons = {
  up: ArrowUpRightIcon,
  down: ArrowDownRightIcon,
  flat: ArrowRightIcon,
} as const;

export const toneClasses = {
  up: "text-success",
  down: "text-destructive",
  flat: "text-muted-foreground",
} as const satisfies Record<ChangeTone, string>;

export function ChangeLabel({
  change,
  changePercent,
  currency,
  suffix,
  className,
}: ChangeLabelProps) {
  const tone = changeTone(change);
  const Icon = toneIcons[tone];

  return (
    <p className={cn("flex items-center gap-1 tabular-nums", toneClasses[tone], className)}>
      <Icon aria-hidden="true" className="size-4 shrink-0" />
      <span>
        {formatSignedMoney(change, currency)}
        {changePercent === null ? null : ` (${formatPercent(changePercent)})`}
      </span>
      {suffix === undefined ? null : <span className="text-muted-foreground">{suffix}</span>}
    </p>
  );
}
