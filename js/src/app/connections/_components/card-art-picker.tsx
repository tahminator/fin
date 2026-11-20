import { CreditCardIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { CardArtResults } from "@/app/connections/_components/card-art-results";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cardSearchName } from "@/lib/card-art";
import { errorMessage } from "@/lib/errors";
import { useSelectCardArt } from "@/lib/queries/card-art";
import type { Account } from "@/lib/types/api";

function CurrentCardArt({ url }: { url: string | null }) {
  return (
    <div className="flex items-center gap-3 rounded-lg border p-2">
      {url === null ? (
        <span className="flex aspect-[1.586] w-20 items-center justify-center rounded-md bg-muted text-muted-foreground">
          <CreditCardIcon className="size-5" aria-hidden="true" />
        </span>
      ) : (
        <img src={url} alt="" className="aspect-[1.586] w-20 rounded-md object-contain" />
      )}
      <p className="text-sm text-muted-foreground">
        {url === null ? "Showing the default icon" : "Current card"}
      </p>
    </div>
  );
}

interface CardArtPickerProps {
  account: Account;
  onDone: () => void;
}

export function CardArtPicker({ account, onDone }: CardArtPickerProps) {
  const [query, setQuery] = useState(() => cardSearchName(account));
  const select = useSelectCardArt();

  const choose = (cardId: string | null) => {
    toast.promise(
      select.mutateAsync({ params: { path: { accountId: account.id } }, body: { cardId } }),
      {
        loading: "Saving card…",
        success: cardId === null ? `${account.name} now uses the default icon` : "Card updated",
        error: (error: unknown) => errorMessage(error),
      },
    );
    onDone();
  };

  return (
    <div className="flex flex-col gap-4">
      <CurrentCardArt url={account.cardArtUrl} />
      <form
        className="flex gap-2"
        onSubmit={(event) => {
          event.preventDefault();

          const value = new FormData(event.currentTarget).get("query");

          setQuery(typeof value === "string" ? value.trim() : "");
        }}
      >
        <Input name="query" defaultValue={query} placeholder="Card name" aria-label="Card name" />
        <Button type="submit" variant="outline">
          Search
        </Button>
      </form>
      <CardArtResults query={query} selectedCardId={account.cardArtId} onChoose={choose} />
      {account.cardArtUrl === null ? null : (
        <Button
          variant="ghost"
          onClick={() => {
            choose(null);
          }}
        >
          Use the default icon instead
        </Button>
      )}
    </div>
  );
}
