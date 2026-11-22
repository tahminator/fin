import { ShieldAlertIcon, ShieldCheckIcon } from "lucide-react";
import { toast } from "sonner";

import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { errorMessage } from "@/lib/errors";
import { useRevokeGrant } from "@/lib/queries/oauth";
import type { OAuthGrant } from "@/lib/types/api";

const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });

function grantSubtitle(grant: OAuthGrant): string {
  const access = grant.scopes.includes("fin:sync") ? "Read + refresh" : "Read only";
  const used =
    grant.lastUsedAt === null
      ? "never used"
      : `used ${dateFormat.format(new Date(grant.lastUsedAt))}`;

  return `${access} · connected ${dateFormat.format(new Date(grant.createdAt))} · ${used}`;
}

function RevokeGrantButton({ grant }: { grant: OAuthGrant }) {
  const revoke = useRevokeGrant();

  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button size="sm" variant="ghost" className="text-destructive" disabled={revoke.isPending}>
          Revoke
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Disconnect {grant.clientName}?</AlertDialogTitle>
          <AlertDialogDescription>
            It loses access immediately. To use it again you’ll need to approve it again.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => {
              toast.promise(revoke.mutateAsync({ params: { path: { grantId: grant.id } } }), {
                loading: `Disconnecting ${grant.clientName}…`,
                success: `${grant.clientName} disconnected`,
                error: (error: unknown) => errorMessage(error),
              });
            }}
          >
            Revoke
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

interface GrantListProps {
  grants: OAuthGrant[];
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => Promise<unknown>;
}

export function GrantList({ grants, hasMore, loadingMore, onLoadMore }: GrantListProps) {
  return (
    <>
      <ListSection header="Apps with access" label="Connected apps">
        {grants.map((grant) => (
          <ListRow key={grant.id}>
            {grant.verified ? (
              <ShieldCheckIcon
                className="size-5 shrink-0 text-muted-foreground"
                aria-label="Verified"
              />
            ) : (
              <ShieldAlertIcon
                className="size-5 shrink-0 text-destructive"
                aria-label="Unverified"
              />
            )}
            <ListRowLabel
              title={
                grant.clientHost === ""
                  ? grant.clientName
                  : `${grant.clientName} · ${grant.clientHost}`
              }
              subtitle={grantSubtitle(grant)}
            />
            <RevokeGrantButton grant={grant} />
          </ListRow>
        ))}
      </ListSection>
      {hasMore ? <LoadMoreButton loading={loadingMore} onLoadMore={onLoadMore} /> : null}
    </>
  );
}

export function GrantListSkeleton() {
  return (
    <ListSection header="Apps with access">
      <ListRow>
        <Skeleton className="h-10 w-full" />
      </ListRow>
    </ListSection>
  );
}
