import { useSearchParams } from "react-router";
import { z } from "zod";

import { ConsentMessage } from "@/app/consent/_components/consent-frame";
import { ConsentView } from "@/app/consent/_components/consent-view";
import { SignInToContinue } from "@/app/consent/_components/sign-in-to-continue";
import type { LoginError } from "@/app/consent/_components/sign-in-to-continue";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { problemStatus } from "@/lib/errors";
import { useMe } from "@/lib/queries/me";
import { useAuthorizationRequest } from "@/lib/queries/oauth";

const requestSchema = z.string().regex(/^[A-Za-z0-9_-]{43}$/u);

const loginErrorSchema = z.enum(["not_allowed", "failed"]);

function readSearch(searchParams: URLSearchParams): { request: string; error: LoginError | null } {
  const request = requestSchema.safeParse(searchParams.get("request"));
  const error = loginErrorSchema.safeParse(searchParams.get("error"));

  return {
    request: request.success ? request.data : "",
    error: error.success ? error.data : null,
  };
}

function ConsentRequestLoader({ request }: { request: string }) {
  const authorization = useAuthorizationRequest(request);

  if (authorization.isPending) {
    return <LoadingState />;
  }

  if (authorization.isError) {
    if (problemStatus(authorization.error) === 404) {
      return (
        <ConsentMessage
          title="This request expired"
          description="Requests last 10 minutes and can be answered once. Start again from your app."
        />
      );
    }

    return <ErrorState error={authorization.error} onRetry={authorization.refetch} />;
  }

  return <ConsentView request={request} authorization={authorization.data} />;
}

export function ConsentPage() {
  const [searchParams] = useSearchParams();
  const { request, error } = readSearch(searchParams);
  const me = useMe();

  if (request === "") {
    return (
      <ConsentMessage
        title="Nothing to approve"
        description="This page opens when an app asks to connect to fin. Start from your app."
      />
    );
  }

  if (me.isPending) {
    return <LoadingState />;
  }

  if (me.isError) {
    if (problemStatus(me.error) === 401) {
      return <SignInToContinue request={request} error={error} />;
    }

    return <ErrorState error={me.error} onRetry={me.refetch} />;
  }

  return <ConsentRequestLoader request={request} />;
}
