import { Navigate } from "react-router";

import { AppShell } from "@/app/_components/app-shell";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { problemStatus } from "@/lib/errors";
import { useMe } from "@/lib/queries/me";

export function AuthenticatedLayout() {
  const me = useMe();

  if (me.isPending) {
    return <LoadingState className="mx-auto w-full max-w-2xl p-4" />;
  }

  if (me.isError) {
    if (problemStatus(me.error) === 401) {
      return <Navigate to="/login" replace />;
    }

    return <ErrorState error={me.error} onRetry={me.refetch} />;
  }

  return <AppShell />;
}
