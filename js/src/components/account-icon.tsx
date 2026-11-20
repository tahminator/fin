import {
  CircleDollarSignIcon,
  CreditCardIcon,
  LandmarkIcon,
  PiggyBankIcon,
  TrendingUpIcon,
  WalletIcon,
} from "lucide-react";
import { useState } from "react";

import { useCardArt } from "@/lib/card-art";
import type { Account } from "@/lib/types/api";
import { cn } from "@/lib/utils";

type AccountIconAccount = Pick<
  Account,
  "institutionName" | "name" | "officialName" | "subtype" | "type"
>;

interface AccountIconProps {
  account: AccountIconAccount;
  className?: string;
}

function isDebitCardAccount(account: Pick<Account, "type" | "subtype">): boolean {
  return (
    account.type === "depository" && account.subtype?.toLowerCase().includes("checking") === true
  );
}

export function AccountIcon({ account, className }: AccountIconProps) {
  const iconClassName = cn("size-4", className);
  const cardArtUrl = useCardArt(account);
  const [cardArtFailed, setCardArtFailed] = useState(false);

  if (cardArtUrl !== null && !cardArtFailed) {
    return (
      <img
        src={cardArtUrl}
        alt=""
        loading="lazy"
        className={cn("rounded-sm object-cover", iconClassName)}
        onError={() => {
          setCardArtFailed(true);
        }}
      />
    );
  }

  if (account.type === "credit" || isDebitCardAccount(account)) {
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
