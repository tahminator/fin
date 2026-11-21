import type { Account } from "@/lib/types/api";

export function isCardAccount(account: Pick<Account, "subtype" | "type">): boolean {
  return account.type === "credit";
}

export function supportsCardArt(account: Pick<Account, "type">): boolean {
  return account.type === "credit";
}

export function cardSearchName(account: Pick<Account, "name" | "officialName">): string {
  const officialName = account.officialName?.trim() ?? "";

  return officialName === "" ? account.name.trim() : officialName;
}
