import { useState } from "react";
import { Link, useNavigate } from "react-router";

import { usePlaidLauncher } from "@/app/connections/_hooks/use-plaid-launcher";
import { EmptyState } from "@/components/empty-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { loadLinkSession } from "@/lib/plaid/link-token-store";
import type { LinkSession } from "@/lib/plaid/link-token-store";

function ResumeLink({ session }: { session: LinkSession }) {
  const navigate = useNavigate();

  const launcher = usePlaidLauncher({
    linkToken: session.linkToken,
    flow: session.flow,
    resumingOAuth: true,
    onDone: () => {
      void navigate("/connections");
    },
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Almost there</CardTitle>
        <CardDescription>Your bank sent you back to fin. Finish connecting below.</CardDescription>
      </CardHeader>
      <CardContent>
        <Button
          className="w-full"
          disabled={!launcher.ready || launcher.busy}
          onClick={launcher.launch}
        >
          {launcher.busy ? <Spinner /> : null}
          Continue connecting
        </Button>
      </CardContent>
    </Card>
  );
}

export function OAuthReturnPage() {
  const [session] = useState(loadLinkSession);

  if (session === null) {
    return (
      <EmptyState
        title="Nothing to resume"
        description="Start connecting your bank from the Connections page."
        action={
          <Button asChild>
            <Link to="/connections">Go to connections</Link>
          </Button>
        }
      />
    );
  }

  return <ResumeLink session={session} />;
}
