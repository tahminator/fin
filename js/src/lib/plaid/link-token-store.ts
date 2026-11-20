import { z } from "zod";

const storageKey = "fin.plaid-link";

const linkSessionSchema = z.object({
  linkToken: z.string().min(1),
  connectionId: z.string().nullable(),
});

export type LinkSession = z.infer<typeof linkSessionSchema>;

export function saveLinkSession(session: LinkSession): void {
  localStorage.setItem(storageKey, JSON.stringify(session));
}

export function loadLinkSession(): LinkSession | null {
  const stored = localStorage.getItem(storageKey);

  if (stored === null) {
    return null;
  }

  const session = linkSessionSchema.safeParse(JSON.parse(stored));

  return session.success ? session.data : null;
}

export function clearLinkSession(): void {
  localStorage.removeItem(storageKey);
}
