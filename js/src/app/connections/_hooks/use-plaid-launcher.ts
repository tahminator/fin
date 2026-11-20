import { usePlaidLink } from "react-plaid-link";
import { toast } from "sonner";

import { errorMessage } from "@/lib/errors";
import { clearLinkSession, saveLinkSession } from "@/lib/plaid/link-token-store";
import { useLinkConnection } from "@/lib/queries/connections";
import { useCompleteReauth } from "@/lib/queries/notifications";

interface PlaidLauncherOptions {
  linkToken: string | undefined;
  connectionId: string | null;
  resumingOAuth?: boolean;
  onDone?: () => void;
}

export function usePlaidLauncher({
  linkToken,
  connectionId,
  resumingOAuth = false,
  onDone,
}: PlaidLauncherOptions) {
  const linkConnection = useLinkConnection();
  const completeReauth = useCompleteReauth();

  const complete = async (publicToken: string | null) => {
    if (connectionId !== null) {
      await completeReauth.mutateAsync({ params: { path: { connectionId } } });

      return;
    }

    if (publicToken === null) {
      throw new Error("Plaid did not return a public token.");
    }

    await linkConnection.mutateAsync({ body: { publicToken } });
  };

  const finish = (publicToken: string | null) => {
    clearLinkSession();

    toast.promise(complete(publicToken).then(onDone), {
      loading: "Finishing connection…",
      success: connectionId === null ? "Bank connected" : "Bank reconnected",
      error: (error: unknown) => errorMessage(error),
    });
  };

  const plaid = usePlaidLink({
    token: linkToken ?? null,
    onSuccess: finish,
    onExit: clearLinkSession,
    ...(resumingOAuth ? { receivedRedirectUri: globalThis.location.href } : {}),
  });

  const launch = () => {
    if (linkToken === undefined) {
      return;
    }

    saveLinkSession({ linkToken, connectionId });
    plaid.open();
  };

  return {
    ready: plaid.ready,
    busy: linkConnection.isPending || completeReauth.isPending,
    launch,
  };
}
