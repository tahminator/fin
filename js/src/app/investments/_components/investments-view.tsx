import { AccountSections } from "@/app/investments/_components/account-sections";
import { PortfolioOverview } from "@/app/investments/_components/portfolio-overview";
import { RangeSelector } from "@/app/investments/_components/range-selector";
import { SecurityControls } from "@/app/investments/_components/security-controls";
import { LoadingState } from "@/components/loading-state";
import { useInvestmentSecurities } from "@/lib/queries/investments";
import type { Account, InvestmentRange } from "@/lib/types/api";

function selectedSecurityIds(off: string[], securityIds: string[] | undefined): string[] | null {
  if (off.length === 0) {
    return [];
  }

  const selected = (securityIds ?? []).filter((id) => !off.includes(id));

  return selected.length === 0 ? null : selected;
}

export function InvestmentsView({
  accounts,
  range,
  off,
  update,
}: {
  accounts: Account[];
  range: InvestmentRange;
  off: string[];
  update: (range: InvestmentRange, off: string[]) => void;
}) {
  const securities = useInvestmentSecurities({ excludeAccountId: [] });

  if (off.length > 0 && securities.isPending) {
    return <LoadingState />;
  }

  const securityIds = selectedSecurityIds(
    off,
    securities.data?.map((security) => security.id),
  );

  return (
    <div className="flex flex-col gap-6">
      <RangeSelector
        value={range}
        onChange={(next) => {
          update(next, off);
        }}
      />
      <PortfolioOverview range={range} securityIds={securityIds} />
      <SecurityControls
        off={off}
        onToggle={(securityId, show) => {
          update(range, show ? off.filter((id) => id !== securityId) : [...off, securityId]);
        }}
        onShowAll={() => {
          update(range, []);
        }}
        onHideAll={(ids) => {
          update(range, ids);
        }}
      />
      <AccountSections accounts={accounts} range={range} securityIds={securityIds} />
    </div>
  );
}
