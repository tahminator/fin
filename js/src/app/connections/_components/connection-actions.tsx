import { ListChecksIcon, RefreshCwIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { usePlaidLauncher } from "@/app/connections/_hooks/use-plaid-launcher";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { errorMessage } from "@/lib/errors";
import { useOnceWhen } from "@/lib/hooks/use-once-when";
import { useCreateAccountSelectionLinkToken, useSyncConnection } from "@/lib/queries/connections";
import { useReauthLinkToken } from "@/lib/queries/notifications";

export function ReconnectButton({ connectionId }: { connectionId: string }) {
  const linkToken = useReauthLinkToken(connectionId);
  const launcher = usePlaidLauncher({
    linkToken: linkToken.data?.linkToken,
    flow: { kind: "reauth", connectionId },
  });

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

interface AccountSelectionLauncherProps {
  connectionId: string;
  linkToken: string;
  onClose: () => void;
}

function AccountSelectionLauncher({
  connectionId,
  linkToken,
  onClose,
}: AccountSelectionLauncherProps) {
  const launcher = usePlaidLauncher({
    linkToken,
    flow: { kind: "account-selection", connectionId },
    onDone: onClose,
    onExit: onClose,
  });

  useOnceWhen(launcher.ready, launcher.launch);

  return (
    <Button
      size="sm"
      variant="outline"
      disabled={!launcher.ready || launcher.busy}
      onClick={launcher.launch}
    >
      {launcher.ready && !launcher.busy ? <ListChecksIcon /> : <Spinner />}
      Edit accounts
    </Button>
  );
}

export function EditAccountsButton({ connectionId }: { connectionId: string }) {
  const [linkToken, setLinkToken] = useState<string | null>(null);
  const createLinkToken = useCreateAccountSelectionLinkToken();

  const requestLinkToken = async () => {
    try {
      const created = await createLinkToken.mutateAsync({ params: { path: { connectionId } } });

      setLinkToken(created.linkToken);
    } catch (error: unknown) {
      toast.error(errorMessage(error));
    }
  };

  if (linkToken !== null) {
    return (
      <AccountSelectionLauncher
        connectionId={connectionId}
        linkToken={linkToken}
        onClose={() => {
          setLinkToken(null);
        }}
      />
    );
  }

  return (
    <Button
      size="sm"
      variant="outline"
      disabled={createLinkToken.isPending}
      onClick={() => {
        void requestLinkToken();
      }}
    >
      {createLinkToken.isPending ? <Spinner /> : <ListChecksIcon />}
      Edit accounts
    </Button>
  );
}
