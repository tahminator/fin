import { Link } from "react-router";

import { AccountChart } from "@/app/investments/_components/account-history";
import { AccountHoldings } from "@/app/investments/_components/account-holdings";
import { InstitutionLogo } from "@/components/institution-logo";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMoney } from "@/lib/format";
import type { Account, InvestmentRange } from "@/lib/types/api";

function AccountSection({
  account,
  range,
  securityIds,
}: {
  account: Account;
  range: InvestmentRange;
  securityIds: string[] | null;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-3">
        <InstitutionLogo url={account.institutionLogoUrl} />
        <div className="min-w-0 flex-1">
          <CardTitle className="truncate">
            {account.name}
            {account.mask === null ? null : (
              <span className="text-muted-foreground"> ••{account.mask}</span>
            )}
          </CardTitle>
          <CardDescription className="truncate">
            {account.institutionName ?? "Unknown institution"}
          </CardDescription>
        </div>
        <p className="shrink-0 type-headline tabular-nums">
          {account.currentBalance === null
            ? "—"
            : formatMoney(account.currentBalance, account.currency)}
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <AccountChart accountId={account.id} range={range} securityIds={securityIds} />
        <AccountHoldings accountId={account.id} />
        <Button variant="outline" size="sm" className="self-start" asChild>
          <Link
            to={{
              pathname: "/transactions",
              search: new URLSearchParams({ accountId: account.id }).toString(),
            }}
          >
            Investment activity
          </Link>
        </Button>
      </CardContent>
    </Card>
  );
}

export function AccountSections({
  accounts,
  range,
  securityIds,
}: {
  accounts: Account[];
  range: InvestmentRange;
  securityIds: string[] | null;
}) {
  return (
    <section aria-labelledby="investment-accounts" className="flex flex-col gap-3">
      <h2 id="investment-accounts" className="px-4 type-footnote text-muted-foreground uppercase">
        Accounts
      </h2>
      {accounts.map((account) => (
        <AccountSection
          key={account.id}
          account={account}
          range={range}
          securityIds={securityIds}
        />
      ))}
    </section>
  );
}
