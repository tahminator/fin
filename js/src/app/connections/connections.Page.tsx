import { ConnectBankButton } from "@/app/connections/_components/connect-bank-button";
import {
  ConnectionList,
  ConnectionListSkeleton,
} from "@/app/connections/_components/connection-list";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { PageHeader } from "@/components/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { useConnections } from "@/lib/queries/connections";
import { useSummary } from "@/lib/queries/summary";
import { useVersion } from "@/lib/queries/version";

function ConnectionResults() {
  const connections = useConnections();

  if (connections.isPending) {
    return <ConnectionListSkeleton />;
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

  return (
    <ListSection header="Overview">
      <ListRow>
        <ListRowLabel title="Accounts" />
        {summary.isSuccess ? (
          <span className="type-headline tabular-nums">{summary.data.accountCount}</span>
        ) : (
          <Skeleton className="h-6 w-8" />
        )}
      </ListRow>
      <ListRow>
        <ListRowLabel title="Connections" />
        {summary.isSuccess ? (
          <span className="type-headline tabular-nums">
            {summary.data.connectionCount} <span className="text-muted-foreground">of 10</span>
          </span>
        ) : (
          <Skeleton className="h-6 w-16" />
        )}
      </ListRow>
    </ListSection>
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
      <ConnectionUsage />
      <ConnectionResults />
      <AppVersion />
    </>
  );
}
