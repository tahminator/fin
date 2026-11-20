import { Link } from "react-router";

import { LoadMoreButton } from "@/components/load-more-button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMoney } from "@/lib/format";
import type { Account } from "@/lib/types/api";

function AccountRow({ account }: { account: Account }) {
  const details = [account.institutionName, account.subtype ?? account.type]
    .filter((detail) => detail !== null)
    .join(" · ");

  return (
    <li>
      <Link
        to={{
          pathname: "/transactions",
          search: new URLSearchParams({ accountId: account.id }).toString(),
        }}
        className="flex items-center justify-between gap-4 rounded-lg px-2 py-3 hover:bg-muted"
      >
        <div className="min-w-0">
          <p className="truncate font-medium">
            {account.name}
            {account.mask === null ? null : (
              <span className="text-muted-foreground"> ••{account.mask}</span>
            )}
          </p>
          <p className="truncate text-sm text-muted-foreground capitalize">{details}</p>
        </div>
        <p className="shrink-0 font-medium tabular-nums">
          {account.currentBalance === null
            ? "—"
            : formatMoney(account.currentBalance, account.currency)}
        </p>
      </Link>
    </li>
  );
}

interface AccountListProps {
  accounts: Account[];
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => Promise<unknown>;
}

export function AccountList({ accounts, hasMore, loadingMore, onLoadMore }: AccountListProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Accounts</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <ul className="divide-y">
          {accounts.map((account) => (
            <AccountRow key={account.id} account={account} />
          ))}
        </ul>
        {hasMore ? <LoadMoreButton loading={loadingMore} onLoadMore={onLoadMore} /> : null}
      </CardContent>
    </Card>
  );
}
