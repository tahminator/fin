import { Navigate } from "react-router";

import { AppShell } from "@/app/_components/app-shell";
import { ErrorState } from "@/components/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { problemStatus } from "@/lib/errors";
import { useMe } from "@/lib/queries/me";

function AuthenticatedLayoutSkeleton() {
  return (
    <div className="flex min-h-dvh flex-col" aria-label="Loading app">
      <div className="mx-auto flex w-full max-w-2xl items-center justify-between px-4 py-3">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="size-8 rounded-full" />
      </div>
      <main className="mx-auto flex w-full max-w-2xl flex-1 flex-col gap-6 px-4 pb-24">
        <Skeleton className="h-32 w-full rounded-[26px]" />
        <Skeleton className="h-80 w-full rounded-[26px]" />
      </main>
    </div>
  );
}

export function AuthenticatedLayout() {
  const me = useMe();

  if (me.isPending) {
    return <AuthenticatedLayoutSkeleton />;
  }

  if (me.isError) {
    if (problemStatus(me.error) === 401) {
      return <Navigate to="/login" replace />;
    }

    return <ErrorState error={me.error} onRetry={me.refetch} />;
  }

  return <AppShell />;
}
