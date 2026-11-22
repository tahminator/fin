import { useState } from "react";
import { toast } from "sonner";

import { ConsentFrame } from "@/app/consent/_components/consent-frame";
import { ScopeList } from "@/app/consent/_components/scope-list";
import { VerificationBadge } from "@/app/consent/_components/verification-badge";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { errorMessage } from "@/lib/errors";
import { useApproveAuthorization, useDenyAuthorization } from "@/lib/queries/oauth";
import type { ConsentRequest, OAuthScope } from "@/lib/types/api";

interface ConsentViewProps {
  request: string;
  authorization: ConsentRequest;
}

function returnToApp(redirectTo: string) {
  globalThis.location.assign(redirectTo);
}

function showFailure(title: string) {
  return (error: unknown) => {
    toast.error(title, { description: errorMessage(error) });
  };
}

export function ConsentView({ request, authorization }: ConsentViewProps) {
  const syncRequested = authorization.scopes.includes("fin:sync");
  const [allowSync, setAllowSync] = useState(syncRequested);
  const approve = useApproveAuthorization();
  const deny = useDenyAuthorization();
  const busy = approve.isPending || deny.isPending || approve.isSuccess || deny.isSuccess;
  const path = { requestId: request };

  const onApprove = () => {
    const scopes: OAuthScope[] = allowSync ? ["fin:read", "fin:sync"] : ["fin:read"];

    approve.mutate(
      { params: { path }, body: { scopes } },
      {
        onSuccess: (data) => {
          returnToApp(data.redirectTo);
        },
        onError: showFailure("Couldn’t approve"),
      },
    );
  };

  const onDeny = () => {
    deny.mutate(
      { params: { path } },
      {
        onSuccess: (data) => {
          returnToApp(data.redirectTo);
        },
        onError: showFailure("Couldn’t deny"),
      },
    );
  };

  return (
    <ConsentFrame
      title={`${authorization.clientName} wants to access fin`}
      subtitle={<VerificationBadge authorization={authorization} />}
    >
      <ScopeList
        syncRequested={syncRequested}
        allowSync={allowSync}
        onAllowSyncChange={setAllowSync}
      />

      <p className="px-1 type-footnote text-muted-foreground">
        It cannot connect, reconnect or remove banks, and it cannot change anything in fin. Your
        data will be sent to this app and the AI model it uses. After you choose, you return to{" "}
        <span className="font-medium text-foreground">{authorization.redirectHost}</span>.
        {authorization.verified
          ? null
          : " This app registered itself, so fin can’t confirm who made it. Only approve it if you just started connecting it."}
      </p>

      <div className="grid grid-cols-2 gap-3">
        <Button size="lg" variant="outline" disabled={busy} onClick={onDeny}>
          {deny.isPending ? <Spinner /> : null}
          Deny
        </Button>
        <Button size="lg" disabled={busy} onClick={onApprove}>
          {approve.isPending ? <Spinner /> : null}
          Approve
        </Button>
      </div>
    </ConsentFrame>
  );
}
