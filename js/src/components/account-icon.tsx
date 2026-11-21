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

function getDepositoryIcon(account: AccountIconAccount, iconClassName: string) {
  const subtype = account.subtype?.toLowerCase() ?? "";

  if (subtype.includes("checking")) {
    return <LandmarkIcon className={iconClassName} aria-hidden="true" />;
  }

  if (subtype.includes("saving")) {
    return <PiggyBankIcon className={iconClassName} aria-hidden="true" />;
  }

  return <WalletIcon className={iconClassName} aria-hidden="true" />;
}

export function AccountIcon({ account, className }: AccountIconProps) {
  const iconClassName = cn("size-4", className);
  const [failedCardArtUrl, setFailedCardArtUrl] = useState<string | null>(null);
  const { cardArtUrl } = account;

  if (account.type === "credit" && cardArtUrl !== null && cardArtUrl !== failedCardArtUrl) {
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
    return getDepositoryIcon(account, iconClassName);
  }

  return <CircleDollarSignIcon className={iconClassName} aria-hidden="true" />;
}
