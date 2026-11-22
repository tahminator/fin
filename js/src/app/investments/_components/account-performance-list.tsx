import { Link } from "react-router";

import { toneClasses } from "@/app/investments/_components/change-label";
import { accountLink } from "@/app/investments/_components/investments-search";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { changeTone, formatMoney, formatPercent, formatSignedMoney } from "@/lib/format";
import type { AccountPerformance, PortfolioRange } from "@/lib/types/api";
import { cn } from "@/lib/utils";

interface AccountPerformanceListProps {
  accounts: AccountPerformance[];
  currency: string;
  range: PortfolioRange;
}

export function AccountPerformanceList({ accounts, currency, range }: AccountPerformanceListProps) {
  if (accounts.length === 0) {
    return null;
  }

  return (
    <ListSection header="Accounts">
      {accounts.map((account) => (
        <ListRow key={account.accountId} className="p-0">
          <Link
            to={accountLink(range, account.accountId)}
            state={{ title: account.name }}
            preventScrollReset
            className="flex min-h-15 min-w-0 flex-1 items-center gap-3 px-4 py-2 outline-none focus-visible:bg-accent active:bg-fill"
          >
            <ListRowLabel
              title={account.name}
              subtitle={
                <span className={cn("tabular-nums", toneClasses[changeTone(account.change)])}>
                  {formatSignedMoney(account.change, currency)}
                  {account.changePercent === null
                    ? null
                    : ` (${formatPercent(account.changePercent)})`}
                </span>
              }
            />
            <p className="shrink-0 type-body tabular-nums">
              {formatMoney(account.value, currency)}
            </p>
          </Link>
        </ListRow>
      ))}
    </ListSection>
  );
}
