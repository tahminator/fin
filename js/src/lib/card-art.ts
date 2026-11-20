import { useQuery } from "@tanstack/react-query";

import type { Account } from "@/lib/types/api";

const searchURL = "https://opencard-db.davidweng.workers.dev/v1/search";
const placeholderImage = "/v1/assets/default-card.webp";
const cacheTime = 7 * 24 * 60 * 60 * 1000;

interface CardImage {
  url: string;
}

interface CardRecord {
  name: string;
  issuer: string | null;
  image: CardImage | null;
}

interface SearchResponse {
  data: CardRecord[];
}

type CardArtAccount = Pick<
  Account,
  "institutionName" | "name" | "officialName" | "subtype" | "type"
>;

function isCardAccount(account: CardArtAccount): boolean {
  return (
    account.type === "credit" ||
    (account.type === "depository" && account.subtype?.toLowerCase().includes("checking") === true)
  );
}

function normalize(value: string): string {
  return value
    .toLowerCase()
    .replaceAll(/[®™]/gu, "")
    .replaceAll(/[^a-z0-9]+/gu, " ")
    .trim();
}

function searchName(account: CardArtAccount): string {
  return account.officialName?.trim() ?? account.name;
}

function cardNameAliases(name: string): string[] {
  const normalized = normalize(name);
  const withoutSupplementary = normalized
    .replaceAll(/\b(?:additional|authorized user|supplementary)\b/gu, "")
    .replaceAll(/\s+/gu, " ")
    .trim();

  return withoutSupplementary === normalized ? [normalized] : [normalized, withoutSupplementary];
}

function cardSearchTerms(name: string): string[] {
  return [name, ...cardNameAliases(name).slice(1)];
}

function hasArtwork(card: CardRecord): boolean {
  const url = card.image?.url;

  return url !== undefined && !url.endsWith(placeholderImage);
}

function scoreAlias(alias: string, candidate: string): number {
  const aliasWords = new Set(alias.split(" "));
  const matchingWords = candidate.split(" ").filter((word) => aliasWords.has(word)).length;
  const wordScore = matchingWords * 100 - Math.abs(candidate.length - alias.length);
  const containsScore = candidate.includes(alias) || alias.includes(candidate) ? 700 : 0;
  const exactScore = candidate === alias ? 1000 : 0;

  return Math.max(exactScore, containsScore, wordScore);
}

function scoreName(account: CardArtAccount, card: CardRecord): number {
  const aliases = cardNameAliases(searchName(account));
  const candidate = normalize(card.name);

  return Math.max(...aliases.map((alias) => scoreAlias(alias, candidate)));
}

function scoreIssuer(account: CardArtAccount, card: CardRecord): number {
  const issuer = normalize(account.institutionName ?? "");
  const candidateIssuer = normalize(card.issuer ?? "");
  const matches =
    issuer !== "" &&
    candidateIssuer !== "" &&
    (candidateIssuer.includes(issuer) || issuer.includes(candidateIssuer));

  return matches ? 50 : 0;
}

function scoreCard(account: CardArtAccount, card: CardRecord): number {
  return scoreName(account, card) + scoreIssuer(account, card);
}

function findArtwork(account: CardArtAccount, cards: CardRecord[]): string | null {
  const card = cards
    .filter((candidate) => hasArtwork(candidate))
    .toSorted((left, right) => scoreCard(account, right) - scoreCard(account, left))
    .at(0);

  return card?.image?.url ?? null;
}

function isCardRecord(value: unknown): value is CardRecord {
  if (
    typeof value !== "object" ||
    value === null ||
    !("name" in value) ||
    typeof value.name !== "string"
  ) {
    return false;
  }

  if (!("issuer" in value) || (value.issuer !== null && typeof value.issuer !== "string")) {
    return false;
  }

  if (!("image" in value)) {
    return false;
  }

  if (value.image === null) {
    return true;
  }

  return (
    typeof value.image === "object" && "url" in value.image && typeof value.image.url === "string"
  );
}

function isSearchResponse(value: unknown): value is SearchResponse {
  if (
    typeof value !== "object" ||
    value === null ||
    !("data" in value) ||
    !Array.isArray(value.data)
  ) {
    return false;
  }

  return value.data.every((card: unknown) => isCardRecord(card));
}

async function searchCards(
  searchTerm: string,
  signal: AbortSignal,
): Promise<SearchResponse | null> {
  const response = await fetch(`${searchURL}?q=${encodeURIComponent(searchTerm)}`, {
    headers: { "X-Client-Name": "fin" },
    signal,
  });

  if (!response.ok) {
    return null;
  }

  const payload: unknown = await response.json();

  return isSearchResponse(payload) ? payload : null;
}

export function useCardArt(account: CardArtAccount): string | null {
  const name = searchName(account);
  const query = useQuery({
    queryKey: ["card-art", account.institutionName, name],
    enabled: isCardAccount(account) && name.length > 0,
    staleTime: cacheTime,
    gcTime: cacheTime,
    queryFn: async ({ signal }) => {
      const responses = await Promise.all(
        cardSearchTerms(name).map((searchTerm) => searchCards(searchTerm, signal)),
      );
      const cards = responses.flatMap((response) => response?.data ?? []);

      return findArtwork(account, cards);
    },
  });

  return query.data ?? null;
}
