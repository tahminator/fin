import { useState } from "react";
import { toast } from "sonner";

import { usePlaidLauncher } from "@/app/connections/_hooks/use-plaid-launcher";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { errorMessage } from "@/lib/errors";
import { useOnceWhen } from "@/lib/hooks/use-once-when";
import { useCreateInvestmentsConsentLinkToken } from "@/lib/queries/connections";

interface ConsentLauncherProps {
  connectionId: string;
  linkToken: string;
  onClose: () => void;
}

function ConsentLauncher({ connectionId, linkToken, onClose }: ConsentLauncherProps) {
  const launcher = usePlaidLauncher({
    linkToken,
    flow: { kind: "investments-consent", connectionId },
    onDone: onClose,
    onExit: onClose,
  });

  useOnceWhen(launcher.ready, launcher.launch);

  return (
    <Button size="sm" disabled={!launcher.ready || launcher.busy} onClick={launcher.launch}>
      <Spinner />
      Share investments
    </Button>
  );
}

export function InvestmentsConsentButton({ connectionId }: { connectionId: string }) {
  const [linkToken, setLinkToken] = useState<string | null>(null);
  const createLinkToken = useCreateInvestmentsConsentLinkToken();

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
      <ConsentLauncher
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
      disabled={createLinkToken.isPending}
      onClick={() => {
        void requestLinkToken();
      }}
    >
      {createLinkToken.isPending ? <Spinner /> : null}
      Share investments
    </Button>
  );
}
