import { usePlaidLink } from "react-plaid-link";
import { toast } from "sonner";

import { errorMessage } from "@/lib/errors";
import { clearLinkSession, saveLinkSession } from "@/lib/plaid/link-token-store";
import type { LinkFlow } from "@/lib/plaid/link-token-store";
import {
  useCompleteAccountSelection,
  useCompleteInvestmentsConsent,
  useLinkConnection,
} from "@/lib/queries/connections";
import { useCompleteReauth } from "@/lib/queries/notifications";

interface PlaidLauncherOptions {
  linkToken: string | undefined;
  flow: LinkFlow;
  resumingOAuth?: boolean;
  onDone?: () => void;
  onExit?: () => void;
}

const successMessages: Record<LinkFlow["kind"], string> = {
  connect: "Bank connected",
  reauth: "Bank reconnected",
  "account-selection": "Accounts updated",
  "investments-consent": "Investments shared",
};

export function usePlaidLauncher({
  linkToken,
  flow,
  resumingOAuth = false,
  onDone,
  onExit,
}: PlaidLauncherOptions) {
  const linkConnection = useLinkConnection();
  const completeReauth = useCompleteReauth();
  const completeAccountSelection = useCompleteAccountSelection();
  const completeInvestmentsConsent = useCompleteInvestmentsConsent();

  const complete = async (publicToken: string | null) => {
    switch (flow.kind) {
      case "connect": {
        if (publicToken === null) {
          throw new Error("Plaid did not return a public token.");
        }

        await linkConnection.mutateAsync({ body: { publicToken } });

        return;
      }
      case "reauth": {
        await completeReauth.mutateAsync({ params: { path: { connectionId: flow.connectionId } } });

        return;
      }
      case "account-selection": {
        await completeAccountSelection.mutateAsync({
          params: { path: { connectionId: flow.connectionId } },
        });

        return;
      }
      case "investments-consent": {
        await completeInvestmentsConsent.mutateAsync({
          params: { path: { connectionId: flow.connectionId } },
        });
      }
    }
  };

  const finish = (publicToken: string | null) => {
    clearLinkSession();

    toast.promise(complete(publicToken).then(onDone), {
      loading: "Finishing connection…",
      success: successMessages[flow.kind],
      error: (error: unknown) => errorMessage(error),
    });
  };

  const exit = () => {
    clearLinkSession();
    onExit?.();
  };

  const plaid = usePlaidLink({
    token: linkToken ?? null,
    onSuccess: finish,
    onExit: exit,
    ...(resumingOAuth ? { receivedRedirectUri: globalThis.location.href } : {}),
  });

  const launch = () => {
    if (linkToken === undefined) {
      return;
    }

    saveLinkSession({ linkToken, flow });
    plaid.open();
  };

  return {
    ready: plaid.ready,
    busy:
      linkConnection.isPending ||
      completeReauth.isPending ||
      completeAccountSelection.isPending ||
      completeInvestmentsConsent.isPending,
    launch,
  };
}
