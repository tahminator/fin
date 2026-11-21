import { ConnectionAccountRow } from "@/app/connections/_components/connection-account-row";
import { ListRow } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { errorMessage } from "@/lib/errors";
import { useConnectionAccounts } from "@/lib/queries/accounts";

export function ConnectionAccounts({ connectionId }: { connectionId: string }) {
  const accounts = useConnectionAccounts(connectionId);

  if (accounts.isPending) {
    return (
      <>
        <ListRow>
          <Skeleton className="size-8 rounded-full" />
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <Skeleton className="h-4 w-36" />
            <Skeleton className="h-3 w-20" />
          </div>
          <Skeleton className="h-4 w-16" />
        </ListRow>
        <ListRow>
          <Skeleton className="size-8 rounded-full" />
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <Skeleton className="h-4 w-28" />
            <Skeleton className="h-3 w-24" />
          </div>
          <Skeleton className="h-4 w-16" />
        </ListRow>
      </>
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
