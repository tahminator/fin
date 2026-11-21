import { ConnectionSection } from "@/app/connections/_components/connection-section";
import { ListRow, ListSection } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import { Skeleton } from "@/components/ui/skeleton";
import type { Connection } from "@/lib/types/api";

function ConnectionSectionSkeleton() {
  return (
    <ListSection>
      <ListRow>
        <Skeleton className="size-8 rounded-full" />
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-3 w-24" />
        </div>
      </ListRow>
      <ListRow>
        <Skeleton className="size-8 rounded-full" />
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3 w-20" />
        </div>
        <Skeleton className="h-4 w-16" />
      </ListRow>
      <ListRow>
        <Skeleton className="h-8 w-20" />
        <Skeleton className="h-8 w-24" />
        <Skeleton className="ml-auto h-8 w-8" />
      </ListRow>
    </ListSection>
  );
}

export function ConnectionListSkeleton() {
  return (
    <div className="flex flex-col gap-6" aria-label="Loading connections">
      <ConnectionSectionSkeleton />
      <ConnectionSectionSkeleton />
    </div>
  );
}

interface ConnectionListProps {
  connections: Connection[];
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => Promise<unknown>;
}

export function ConnectionList({
  connections,
  hasMore,
  loadingMore,
  onLoadMore,
}: ConnectionListProps) {
  return (
    <div className="flex flex-col gap-6">
      {connections.map((connection) => (
        <ConnectionSection key={connection.id} connection={connection} />
      ))}
      {hasMore ? (
        <ListSection>
          <ListRow className="p-0">
            <LoadMoreButton loading={loadingMore} onLoadMore={onLoadMore} />
          </ListRow>
        </ListSection>
      ) : null}
    </div>
  );
}
