import { EyeIcon, EyeOffIcon } from "lucide-react";
import { Link } from "react-router";

import { AccountIcon } from "@/components/account-icon";
import { InstitutionLogo } from "@/components/institution-logo";
import { LoadMoreButton } from "@/components/load-more-button";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMoney } from "@/lib/format";
import type { Account } from "@/lib/types/api";
import { cn } from "@/lib/utils";

interface AccountRowProps {
  account: Account;
  hidden: boolean;
  onToggle: (accountId: string) => void;
}

function AccountRow({ account, hidden, onToggle }: AccountRowProps) {
  const details = [account.institutionName, account.subtype ?? account.type]
    .filter((detail) => detail !== null)
    .join(" · ");

  return (
    <li className={cn("flex items-center gap-1", hidden ? "opacity-50" : null)}>
      <Link
        to={{
          pathname: "/transactions",
          search: new URLSearchParams({ accountId: account.id }).toString(),
        }}
        className="flex min-w-0 flex-1 items-center justify-between gap-4 rounded-lg px-2 py-3 hover:bg-muted"
      >
        <div className="flex min-w-0 items-center gap-3">
          <div className="relative shrink-0">
            <InstitutionLogo url={account.institutionLogoUrl} />
            <span className="absolute -right-1 -bottom-1 flex size-4 items-center justify-center rounded-full border bg-background">
              <AccountIcon account={account} className="size-2.5" />
            </span>
          </div>
          <div className="min-w-0">
            <p className="truncate font-medium">
              {account.name}
              {account.mask === null ? null : (
                <span className="text-muted-foreground"> ••{account.mask}</span>
              )}
            </p>
            <p className="truncate text-sm text-muted-foreground capitalize">{details}</p>
          </div>
        </div>
        <p className="shrink-0 font-medium tabular-nums">
          {account.currentBalance === null
            ? "—"
            : formatMoney(account.currentBalance, account.currency)}
        </p>
      </Link>
      <Button
        variant="ghost"
        size="icon"
        className="shrink-0"
        aria-label={hidden ? `Show ${account.name}` : `Hide ${account.name}`}
        aria-pressed={hidden}
        onClick={() => {
          onToggle(account.id);
        }}
      >
        {hidden ? <EyeOffIcon /> : <EyeIcon />}
      </Button>
    </li>
  );
}

interface AccountListProps {
  accounts: Account[];
  hiddenAccountIds: string[];
  onToggleAccount: (accountId: string) => void;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => Promise<unknown>;
}

export function AccountList({
  accounts,
  hiddenAccountIds,
  onToggleAccount,
  hasMore,
  loadingMore,
  onLoadMore,
}: AccountListProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Accounts</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <ul className="divide-y">
          {accounts.map((account) => (
            <AccountRow
              key={account.id}
              account={account}
              hidden={hiddenAccountIds.includes(account.id)}
              onToggle={onToggleAccount}
            />
          ))}
        </ul>
        {hasMore ? <LoadMoreButton loading={loadingMore} onLoadMore={onLoadMore} /> : null}
      </CardContent>
    </Card>
  );
}
