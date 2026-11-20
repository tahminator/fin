import { ConnectionAccounts } from "@/app/connections/_components/connection-accounts";
import {
  EditAccountsButton,
  ReconnectButton,
  SyncButton,
} from "@/app/connections/_components/connection-actions";
import { RemoveConnectionButton } from "@/app/connections/_components/remove-connection-button";
import { InstitutionLogo } from "@/components/institution-logo";
import { LoadMoreButton } from "@/components/load-more-button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
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

function ConnectionRow({ connection }: { connection: Connection }) {
  const name = connection.institutionName ?? "Unknown bank";
  const badge = statusBadges[connection.status];

  return (
    <li className="flex flex-col gap-3 py-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex min-w-0 items-start gap-3">
          <InstitutionLogo url={connection.institutionLogoUrl} />
          <div className="min-w-0">
            <p className="truncate font-medium">{name}</p>
            <p className="text-sm text-muted-foreground">{statusLine(connection)}</p>
            {connection.lastError === null ? null : (
              <p className="text-sm text-destructive">{connection.lastError}</p>
            )}
          </div>
        </div>
        <Badge variant={badge.variant}>{badge.label}</Badge>
      </div>
      <ConnectionAccounts connectionId={connection.id} />
      <div className="flex items-center gap-2">
        {connection.status === "needs_reauth" ? (
          <ReconnectButton connectionId={connection.id} />
        ) : (
          <>
            <SyncButton connectionId={connection.id} />
            <EditAccountsButton connectionId={connection.id} />
          </>
        )}
        <RemoveConnectionButton connectionId={connection.id} name={name} />
      </div>
    </li>
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
    <Card>
      <CardContent className="flex flex-col gap-2">
        <ul className="divide-y">
          {connections.map((connection) => (
            <ConnectionRow key={connection.id} connection={connection} />
          ))}
        </ul>
        {hasMore ? <LoadMoreButton loading={loadingMore} onLoadMore={onLoadMore} /> : null}
      </CardContent>
    </Card>
  );
}
