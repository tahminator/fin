import { EyeIcon, EyeOffIcon } from "lucide-react";
import { Link } from "react-router";

import { AccountIcon } from "@/components/account-icon";
import { InstitutionLogo } from "@/components/institution-logo";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import { Button } from "@/components/ui/button";
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
    <ListRow className="gap-0 py-0 pr-1 pl-0">
      <Link
        to={{
          pathname: "/transactions",
          search: new URLSearchParams({ accountId: account.id }).toString(),
        }}
        className={cn(
          "flex min-h-15 min-w-0 flex-1 items-center gap-3 py-2 pl-4 transition-[background-color,opacity] outline-none focus-visible:bg-accent active:bg-fill",
          hidden ? "opacity-40" : null,
        )}
      >
        <div className="relative shrink-0">
          <InstitutionLogo url={account.institutionLogoUrl} />
          <span className="absolute -right-1 -bottom-1 flex size-4 items-center justify-center rounded-full bg-card ring-2 ring-card">
            <AccountIcon account={account} className="size-2.5" />
          </span>
        </div>
        <ListRowLabel
          title={
            <>
              {account.name}
              {account.mask === null ? null : (
                <span className="text-muted-foreground"> ••{account.mask}</span>
              )}
            </>
          }
          subtitle={<span className="capitalize">{details}</span>}
        />
        <p className="shrink-0 type-body tabular-nums">
          {account.currentBalance === null
            ? "—"
            : formatMoney(account.currentBalance, account.currency)}
        </p>
      </Link>
      <Button
        variant="ghost"
        size="icon"
        className="ml-1 shrink-0"
        aria-label={hidden ? `Show ${account.name}` : `Hide ${account.name}`}
        aria-pressed={hidden}
        onClick={() => {
          onToggle(account.id);
        }}
      >
        {hidden ? <EyeOffIcon /> : <EyeIcon />}
      </Button>
    </ListRow>
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
    <ListSection header="Accounts" footer="Hidden accounts are left out of the totals above.">
      {accounts.map((account) => (
        <AccountRow
          key={account.id}
          account={account}
          hidden={hiddenAccountIds.includes(account.id)}
          onToggle={onToggleAccount}
        />
      ))}
      {hasMore ? (
        <ListRow className="p-0">
          <LoadMoreButton loading={loadingMore} onLoadMore={onLoadMore} />
        </ListRow>
      ) : null}
    </ListSection>
  );
}
