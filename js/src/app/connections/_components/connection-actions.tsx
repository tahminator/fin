import { RefreshCwIcon } from "lucide-react";
import { toast } from "sonner";

import { usePlaidLauncher } from "@/app/connections/_hooks/use-plaid-launcher";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { errorMessage } from "@/lib/errors";
import { useSyncConnection } from "@/lib/queries/connections";
import { useReauthLinkToken } from "@/lib/queries/notifications";

export function ReconnectButton({ connectionId }: { connectionId: string }) {
  const linkToken = useReauthLinkToken(connectionId);
  const launcher = usePlaidLauncher({ linkToken: linkToken.data?.linkToken, connectionId });

  return (
    <Button size="sm" disabled={!launcher.ready || launcher.busy} onClick={launcher.launch}>
      {launcher.busy ? <Spinner /> : null}
      Reconnect
    </Button>
  );
}

export function SyncButton({ connectionId }: { connectionId: string }) {
  const sync = useSyncConnection();

  return (
    <Button
      size="sm"
      variant="outline"
      disabled={sync.isPending}
      onClick={() => {
        const syncing = sync
          .mutateAsync({ params: { path: { connectionId } } })
          .then((connection) => {
            if (connection.status === "needs_reauth") {
              throw new Error("This bank needs you to sign in again. Use Reconnect.");
            }

            return connection;
          });

        toast.promise(syncing, {
          loading: "Syncing…",
          success: "Connection synced",
          error: (error: unknown) => errorMessage(error),
        });
      }}
    >
      {sync.isPending ? <Spinner /> : <RefreshCwIcon />}
      Sync
    </Button>
  );
}
