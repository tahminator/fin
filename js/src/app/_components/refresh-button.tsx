import { RefreshCwIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { errorMessage } from "@/lib/errors";
import { useSyncAll } from "@/lib/queries/connections";
import type { SyncSummary } from "@/lib/types/api";

function showSyncSummary(summary: SyncSummary) {
  const problems = [
    summary.reauthRequired > 0 ? `${summary.reauthRequired} need sign-in` : null,
    summary.failed > 0 ? `${summary.failed} failed` : null,
  ].filter((problem) => problem !== null);

  const recent =
    summary.throttled > 0
      ? `${summary.throttled} refreshed recently, try again in a few minutes`
      : null;

  const title = `Synced ${summary.synced} connection${summary.synced === 1 ? "" : "s"}`;

  if (summary.synced === 0 && problems.length === 0 && recent !== null) {
    toast.info("Already up to date", { description: recent });

    return;
  }

  if (recent !== null) {
    problems.push(recent);
  }

  if (problems.length === 0) {
    toast.success(title);

    return;
  }

  toast.warning(title, { description: problems.join(", ") });
}

export function RefreshButton() {
  const syncAll = useSyncAll();

  return (
    <Button
      variant="ghost"
      size="icon-lg"
      className="text-foreground"
      aria-label="Refresh all connections"
      disabled={syncAll.isPending}
      onClick={() => {
        syncAll.mutate(
          {},
          {
            onSuccess: showSyncSummary,
            onError: (error) => {
              toast.error("Refresh failed", { description: errorMessage(error) });
            },
          },
        );
      }}
    >
      <RefreshCwIcon className={syncAll.isPending ? "animate-spin" : undefined} />
    </Button>
  );
}
