import { AccountIcon } from "@/components/account-icon";
import { LoadMoreButton } from "@/components/load-more-button";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { errorMessage } from "@/lib/errors";
import { formatMoney } from "@/lib/format";
import { useConnectionAccounts } from "@/lib/queries/accounts";
import type { Account } from "@/lib/types/api";

function ConnectionAccountRow({ account }: { account: Account }) {
  return (
    <li className="flex items-center justify-between gap-4 py-2">
      <div className="flex min-w-0 items-center gap-2">
        <AccountIcon account={account} className="shrink-0 text-muted-foreground" />
        <p className="truncate text-sm">
          {account.name}
          {account.mask === null ? null : (
            <span className="text-muted-foreground"> ••{account.mask}</span>
          )}
          <span className="text-muted-foreground capitalize">
            {" · "}
            {account.subtype ?? account.type}
          </span>
        </p>
      </div>
      <p className="shrink-0 text-sm tabular-nums">
        {account.currentBalance === null
          ? "—"
          : formatMoney(account.currentBalance, account.currency)}
      </p>
    </li>
  );
}

export function ConnectionAccounts({ connectionId }: { connectionId: string }) {
  const accounts = useConnectionAccounts(connectionId);

  if (accounts.isPending) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner />
        Loading accounts…
      </div>
    );
  }

  if (accounts.isError) {
    return (
      <div className="flex items-center justify-between gap-2 text-sm">
        <p className="text-destructive">{errorMessage(accounts.error)}</p>
        <Button
          size="sm"
          variant="outline"
          onClick={() => {
            void accounts.refetch();
          }}
        >
          Try again
        </Button>
      </div>
    );
  }

  if (accounts.data.length === 0) {
    return <p className="text-sm text-muted-foreground">No accounts shared with this connection</p>;
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg border px-3">
      <ul className="divide-y" aria-label="Accounts">
        {accounts.data.map((account) => (
          <ConnectionAccountRow key={account.id} account={account} />
        ))}
      </ul>
      {accounts.hasNextPage ? (
        <LoadMoreButton loading={accounts.isFetchingNextPage} onLoadMore={accounts.fetchNextPage} />
      ) : null}
    </div>
  );
}
