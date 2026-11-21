import { CardArtDialog } from "@/app/connections/_components/card-art-dialog";
import { AccountIcon } from "@/components/account-icon";
import { ListRow, ListRowLabel } from "@/components/list-section";
import { supportsCardArt } from "@/lib/card-art";
import { formatMoney } from "@/lib/format";
import type { Account } from "@/lib/types/api";

export function ConnectionAccountRow({ account }: { account: Account }) {
  return (
    <ListRow>
      <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-secondary text-muted-foreground">
        <AccountIcon account={account} className="size-5" />
      </span>
      <ListRowLabel
        title={
          <>
            {account.name}
            {account.mask === null ? null : (
              <span className="text-muted-foreground"> ••{account.mask}</span>
            )}
          </>
        }
        subtitle={
          <>
            <span className="capitalize">{account.subtype ?? account.type}</span>
            {supportsCardArt(account) ? (
              <>
                {" · "}
                <CardArtDialog account={account} />
              </>
            ) : null}
          </>
        }
      />
      <p className="shrink-0 type-body tabular-nums">
        {account.currentBalance === null
          ? "—"
          : formatMoney(account.currentBalance, account.currency)}
      </p>
    </ListRow>
  );
}
