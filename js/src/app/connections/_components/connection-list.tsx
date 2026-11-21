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
import { formatDateTime } from "@/lib/format";
import type { Connection, ConnectionStatus } from "@/lib/types/api";

const statusBadges: Record<
  ConnectionStatus,
  { label: string; variant: "secondary" | "destructive" }
> = {
  active: { label: "Connected", variant: "secondary" },
  needs_reauth: { label: "Sign-in needed", variant: "destructive" },
};

function statusLine(connection: Connection): string {
  if (connection.status === "needs_reauth") {
    return `Since ${formatDateTime(connection.statusChangedAt)}`;
  }

  if (connection.lastSyncedAt === null) {
    return "Never synced";
  }

  return `Last synced ${formatDateTime(connection.lastSyncedAt)}`;
}

function ConnectionSection({ connection }: { connection: Connection }) {
  const name = connection.institutionName ?? "Unknown bank";
  const badge = statusBadges[connection.status];

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
          subtitle={statusLine(connection)}
        />
        <Badge variant={badge.variant}>{badge.label}</Badge>
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
