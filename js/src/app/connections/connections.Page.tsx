import { ConnectBankButton } from "@/app/connections/_components/connect-bank-button";
import { ConnectionList } from "@/app/connections/_components/connection-list";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { PageHeader } from "@/components/page-header";
import { useConnections } from "@/lib/queries/connections";
import { useSummary } from "@/lib/queries/summary";
import { useVersion } from "@/lib/queries/version";

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

function ConnectionUsage() {
  const summary = useSummary();

  if (!summary.isSuccess) {
    return null;
  }

  return (
    <p className="px-4 type-footnote text-muted-foreground">
      {summary.data.accountCount} accounts · {summary.data.connectionCount} of 10 connections
    </p>
  );
}

function AppVersion() {
  const version = useVersion().data?.version;

  if (version === undefined || version === null) {
    return null;
  }

  return <p className="mt-auto text-center type-footnote text-muted-foreground">{version}</p>;
}

export function ConnectionsPage() {
  return (
    <>
      <PageHeader title="Connections" actions={<ConnectBankButton />} />
      <ConnectionResults />
      <ConnectionUsage />
      <AppVersion />
    </>
  );
}
