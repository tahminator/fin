import { ListRow, ListRowLabel } from "@/components/list-section";
import { Checkbox } from "@/components/ui/checkbox";
import { sumDecimals } from "@/lib/decimal";
import { formatMoney } from "@/lib/format";
import { groupBy } from "@/lib/group-by";
import { securityLabel } from "@/lib/securities";
import type { InvestmentHolding, InvestmentSecurity } from "@/lib/types/api";

function currentValues(holdings: InvestmentHolding[]): string[] {
  const byCurrency = groupBy(holdings, (holding) => holding.currency ?? "");

  return [...byCurrency].map(([currency, held]) =>
    formatMoney(
      sumDecimals(held.map((holding) => holding.institutionValue)),
      currency === "" ? null : currency,
    ),
  );
}

export function SecurityRow({
  security,
  holdings,
  checked,
  onToggle,
}: {
  security: InvestmentSecurity;
  holdings: InvestmentHolding[];
  checked: boolean;
  onToggle: (securityId: string, show: boolean) => void;
}) {
  const label = securityLabel(security);
  const quantity =
    holdings.length === 0 ? null : sumDecimals(holdings.map((holding) => holding.quantity));
  const accounts = [...new Set(holdings.map((holding) => holding.accountName))].join(", ");
  const details = [
    security.type,
    quantity === null ? "Not currently held" : `${Number(quantity).toLocaleString()} units`,
    accounts === "" ? null : accounts,
  ]
    .filter((detail) => detail !== null)
    .join(" · ");

  return (
    <ListRow className="p-0">
      <label className="flex min-h-15 min-w-0 flex-1 cursor-pointer items-center gap-3 py-2 pr-4 pl-4">
        <Checkbox
          checked={checked}
          aria-label={`Include ${label} in the chart`}
          onCheckedChange={(next) => {
            onToggle(security.id, next === true);
          }}
        />
        <ListRowLabel
          title={label}
          subtitle={<span className="first-letter:uppercase">{details}</span>}
        />
        <div className="shrink-0 text-right type-body tabular-nums">
          {currentValues(holdings).map((value) => (
            <p key={value}>{value}</p>
          ))}
        </div>
      </label>
    </ListRow>
  );
}
