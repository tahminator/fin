import { GrantList, GrantListSkeleton } from "@/app/apps/_components/grant-list";
import { McpEndpoint } from "@/app/apps/_components/mcp-endpoint";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { PageHeader } from "@/components/page-header";
import { useGrants } from "@/lib/queries/oauth";

function GrantResults() {
  const grants = useGrants();

  if (grants.isPending) {
    return <GrantListSkeleton />;
  }

  if (grants.isError) {
    return <ErrorState error={grants.error} onRetry={grants.refetch} />;
  }

  if (grants.data.length === 0) {
    return (
      <EmptyState
        title="No apps connected"
        description="Add the URL above to an AI app such as Claude to let it read your finances."
      />
    );
  }

  return (
    <GrantList
      grants={grants.data}
      hasMore={grants.hasNextPage}
      loadingMore={grants.isFetchingNextPage}
      onLoadMore={grants.fetchNextPage}
    />
  );
}

export function AppsPage() {
  return (
    <>
      <PageHeader title="Connected apps" />
      <McpEndpoint />
      <GrantResults />
    </>
  );
}
