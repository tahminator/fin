import { PlusIcon } from "lucide-react";

import { usePlaidLauncher } from "@/app/connections/_hooks/use-plaid-launcher";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { errorMessage } from "@/lib/errors";
import { useLinkToken } from "@/lib/queries/connections";

export function ConnectBankButton() {
  const linkToken = useLinkToken();
  const launcher = usePlaidLauncher({
    linkToken: linkToken.data?.linkToken,
    flow: { kind: "connect" },
  });

  return (
    <div className="flex flex-col items-end gap-1">
      <Button size="sm" disabled={!launcher.ready || launcher.busy} onClick={launcher.launch}>
        {launcher.busy ? <Spinner /> : <PlusIcon />}
        Connect bank
      </Button>
      {linkToken.isError ? (
        <p className="text-xs text-destructive">{errorMessage(linkToken.error)}</p>
      ) : null}
    </div>
  );
}
