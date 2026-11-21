import { ConnectionAccounts } from "@/app/connections/_components/connection-accounts";
import {
  EditAccountsButton,
  ReconnectButton,
  SyncButton,
} from "@/app/connections/_components/connection-actions";
import { RemoveConnectionButton } from "@/app/connections/_components/remove-connection-button";
import { InstitutionLogo } from "@/components/institution-logo";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { formatAge, formatDateTime } from "@/lib/format";
import { useNow } from "@/lib/hooks/use-now";
import type { Connection } from "@/lib/types/api";

function statusLine(connection: Connection, now: number): string {
  if (connection.status === "needs_reauth") {
    return `Since ${formatDateTime(connection.statusChangedAt)}`;
  }

  if (connection.lastSyncedAt === null) {
    return "Never synced";
  }

  return `Synced ${formatAge(connection.lastSyncedAt, now)} ago`;
}

function ConnectionSection({ connection }: { connection: Connection }) {
  const name = connection.institutionName ?? "Unknown bank";
  const now = useNow();

  return (
    <ListSection
      label={name}
      footer={
        connection.lastError === null ? undefined : (
          <span className="text-destructive">{connection.lastError}</span>
        )
      }
    >
      <ListRow>
        <InstitutionLogo url={connection.institutionLogoUrl} />
        <ListRowLabel
          title={<span className="font-semibold">{name}</span>}
          subtitle={statusLine(connection, now)}
        />
        {connection.status === "needs_reauth" ? (
          <Badge variant="destructive">Sign-in needed</Badge>
        ) : null}
      </ListRow>
      <ConnectionAccounts connectionId={connection.id} />
      <ListRow className="gap-2 not-first:before:left-4">
        {connection.status === "needs_reauth" ? (
          <ReconnectButton connectionId={connection.id} />
        ) : (
          <>
            <SyncButton connectionId={connection.id} />
            <EditAccountsButton connectionId={connection.id} />
          </>
        )}
        <RemoveConnectionButton connectionId={connection.id} name={name} />
      </ListRow>
    </ListSection>
  );
}

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
