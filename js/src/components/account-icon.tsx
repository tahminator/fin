import {
  CircleDollarSignIcon,
  CreditCardIcon,
  LandmarkIcon,
  PiggyBankIcon,
  TrendingUpIcon,
  WalletIcon,
} from "lucide-react";

import type { Account } from "@/lib/types/api";
import { cn } from "@/lib/utils";

interface AccountIconProps {
  account: Pick<Account, "type" | "subtype">;
  className?: string;
}

export function AccountIcon({ account, className }: AccountIconProps) {
  const iconClassName = cn("size-4", className);

  if (account.type === "credit") {
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
