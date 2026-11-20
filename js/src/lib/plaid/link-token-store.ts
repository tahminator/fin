import { z } from "zod";

const storageKey = "fin.plaid-link";

const linkFlowSchema = z.discriminatedUnion("kind", [
  z.object({ kind: z.literal("connect") }),
  z.object({ kind: z.literal("reauth"), connectionId: z.string() }),
  z.object({ kind: z.literal("account-selection"), connectionId: z.string() }),
]);

const linkSessionSchema = z.object({
  linkToken: z.string().min(1),
  flow: linkFlowSchema,
});

export type LinkFlow = z.infer<typeof linkFlowSchema>;

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
