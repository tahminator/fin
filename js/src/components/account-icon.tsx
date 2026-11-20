import {
  CircleDollarSignIcon,
  CreditCardIcon,
  LandmarkIcon,
  PiggyBankIcon,
  TrendingUpIcon,
  WalletIcon,
} from "lucide-react";
import { useState } from "react";

import { isCardAccount } from "@/lib/card-art";
import type { Account } from "@/lib/types/api";
import { cn } from "@/lib/utils";

type AccountIconAccount = Pick<Account, "cardArtUrl" | "subtype" | "type">;

interface AccountIconProps {
  account: AccountIconAccount;
  className?: string;
}

export function AccountIcon({ account, className }: AccountIconProps) {
  const iconClassName = cn("size-4", className);
  const [failedCardArtUrl, setFailedCardArtUrl] = useState<string | null>(null);
  const { cardArtUrl } = account;

  if (cardArtUrl !== null && cardArtUrl !== failedCardArtUrl) {
    return (
      <img
        src={cardArtUrl}
        alt=""
        loading="lazy"
        className={cn("rounded-sm object-cover", iconClassName)}
        onError={() => {
          setFailedCardArtUrl(cardArtUrl);
        }}
      />
    );
  }

  if (isCardAccount(account)) {
    return <CreditCardIcon className={iconClassName} aria-hidden="true" />;
  }

  if (account.type === "loan") {
    return <LandmarkIcon className={iconClassName} aria-hidden="true" />;
  }

  if (account.type === "investment") {
    return <TrendingUpIcon className={iconClassName} aria-hidden="true" />;
  }

  if (account.type === "depository") {
    const isSavings = account.subtype !== null && account.subtype.toLowerCase().includes("saving");

    return isSavings ? (
      <PiggyBankIcon className={iconClassName} aria-hidden="true" />
    ) : (
      <WalletIcon className={iconClassName} aria-hidden="true" />
    );
  }

  return <CircleDollarSignIcon className={iconClassName} aria-hidden="true" />;
}
