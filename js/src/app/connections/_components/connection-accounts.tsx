import { ConnectionAccountRow } from "@/app/connections/_components/connection-account-row";
import { ListRow } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { errorMessage } from "@/lib/errors";
import { useConnectionAccounts } from "@/lib/queries/accounts";

export function ConnectionAccounts({ connectionId }: { connectionId: string }) {
  const accounts = useConnectionAccounts(connectionId);

  if (accounts.isPending) {
    return (
      <ListRow className="type-subheadline text-muted-foreground">
        <Spinner />
        Loading accounts…
      </ListRow>
    );
  }

  if (accounts.isError) {
    return (
      <ListRow className="justify-between type-subheadline">
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
      </ListRow>
    );
  }

  if (accounts.data.length === 0) {
    return (
      <ListRow className="type-subheadline text-muted-foreground">
        No accounts shared with this connection
      </ListRow>
    );
  }

  return (
    <>
      {accounts.data.map((account) => (
        <ConnectionAccountRow key={account.id} account={account} />
      ))}
      {accounts.hasNextPage ? (
        <ListRow className="p-0">
          <LoadMoreButton
            loading={accounts.isFetchingNextPage}
            onLoadMore={accounts.fetchNextPage}
          />
        </ListRow>
      ) : null}
    </>
  );
}
