import { ConnectionAccounts } from "@/app/connections/_components/connection-accounts";
import {
  EditAccountsButton,
  ReconnectButton,
  SyncButton,
} from "@/app/connections/_components/connection-actions";
import { RemoveConnectionButton } from "@/app/connections/_components/remove-connection-button";
import { InstitutionLogo } from "@/components/institution-logo";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { Badge } from "@/components/ui/badge";
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

export function ConnectionSection({ connection }: { connection: Connection }) {
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
