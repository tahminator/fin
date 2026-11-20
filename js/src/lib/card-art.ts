import { useQuery } from "@tanstack/react-query";

import type { Account } from "@/lib/types/api";

const searchURL = "https://opencard-db.davidweng.workers.dev/v1/search";
const placeholderImage = "/v1/assets/default-card.webp";
const cacheTime = 7 * 24 * 60 * 60 * 1_000;

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
    .replace(/®|™/g, "")
    .replace(/[^a-z0-9]+/g, " ")
    .trim();
}

function searchName(account: CardArtAccount): string {
  return account.officialName?.trim() || account.name;
}

function cardNameAliases(name: string): string[] {
  const normalized = normalize(name);
  const withoutSupplementary = normalized
    .replace(/\b(additional|authorized user|supplementary)\b/g, "")
    .replace(/\s+/g, " ")
    .trim();

  return withoutSupplementary === normalized ? [normalized] : [normalized, withoutSupplementary];
}

function hasArtwork(card: CardRecord): boolean {
  const url = card.image?.url;

  return url !== undefined && !url.endsWith(placeholderImage);
}

function scoreCard(account: CardArtAccount, card: CardRecord): number {
  const aliases = cardNameAliases(searchName(account));
  const candidate = normalize(card.name);
  const issuer = normalize(account.institutionName ?? "");
  const candidateIssuer = normalize(card.issuer ?? "");
  let score = 0;

  for (const alias of aliases) {
    if (candidate === alias) {
      score = Math.max(score, 1_000);
      continue;
    }

    if (candidate.includes(alias) || alias.includes(candidate)) {
      score = Math.max(score, 700);
    }

    const aliasWords = new Set(alias.split(" "));
    const matchingWords = candidate.split(" ").filter((word) => aliasWords.has(word)).length;
    score = Math.max(score, matchingWords * 100 - Math.abs(candidate.length - alias.length));
  }

  if (issuer !== "" && (candidateIssuer.includes(issuer) || issuer.includes(candidateIssuer))) {
    score += 50;
  }

  return score;
}

function findArtwork(account: CardArtAccount, cards: CardRecord[]): string | null {
  return (
    cards
      .filter(hasArtwork)
      .sort((left, right) => scoreCard(account, right) - scoreCard(account, left))[0]?.image?.url ??
    null
  );
}

export function useCardArt(account: CardArtAccount): string | null {
  const name = searchName(account);
  const query = useQuery({
    queryKey: ["card-art", account.institutionName, name],
    enabled: isCardAccount(account) && name.length > 0,
    staleTime: cacheTime,
    gcTime: cacheTime,
    queryFn: async ({ signal }) => {
      for (const searchTerm of [name, ...cardNameAliases(name)]) {
        const response = await fetch(`${searchURL}?q=${encodeURIComponent(searchTerm)}`, {
          headers: { "X-Client-Name": "fin" },
          signal,
        });

        if (!response.ok) {
          continue;
        }

        const payload = (await response.json()) as SearchResponse;
        const artwork = findArtwork(account, payload.data);

        if (artwork !== null) {
          return artwork;
        }
      }

      return null;
    },
  });

  return query.data ?? null;
}
