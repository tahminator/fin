import { ConnectBankButton } from "@/app/connections/_components/connect-bank-button";
import { ConnectionList } from "@/app/connections/_components/connection-list";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { useConnections } from "@/lib/queries/connections";

function ConnectionResults() {
  const connections = useConnections();

  if (connections.isPending) {
    return <LoadingState />;
  }

  if (connections.isError) {
    return <ErrorState error={connections.error} onRetry={connections.refetch} />;
  }

  if (connections.data.length === 0) {
    return (
      <EmptyState
        title="No banks connected"
        description="Use Connect bank to link your first institution."
      />
    );
  }

  return (
    <ConnectionList
      connections={connections.data}
      hasMore={connections.hasNextPage}
      loadingMore={connections.isFetchingNextPage}
      onLoadMore={connections.fetchNextPage}
    />
  );
}

export function ConnectionsPage() {
  return (
    <>
      <header className="flex items-start justify-between gap-2">
        <h1 className="text-xl font-semibold tracking-tight">Connections</h1>
        <ConnectBankButton />
      </header>
      <ConnectionResults />
    </>
  );
}
