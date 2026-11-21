import { useState } from "react";

import { CardArtPicker } from "@/app/connections/_components/card-art-picker";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import type { Account } from "@/lib/types/api";

function cardArtDescription(account: Account): string {
  if (account.cardArtSource === "selected") {
    return "You picked this card. Search to choose a different one.";
  }

  if (account.cardArtSource === null) {
    return "This card is matched on the next sync. You can also pick it yourself now.";
  }

  return account.cardArtUrl === null
    ? "No matching card was found automatically. Search for yours below."
    : "Matched automatically from the account name. Wrong card? Pick another.";
}

export function CardArtDialog({ account }: { account: Account }) {
  const [open, setOpen] = useState(false);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" variant="link" className="h-auto p-0 align-baseline font-normal">
          Logo
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Card for {account.name}</DialogTitle>
          <DialogDescription>{cardArtDescription(account)}</DialogDescription>
        </DialogHeader>
        <CardArtPicker
          account={account}
          onDone={() => {
            setOpen(false);
          }}
        />
      </DialogContent>
    </Dialog>
  );
}
