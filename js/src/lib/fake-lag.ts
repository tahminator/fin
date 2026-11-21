export const fakeLagHeader = "X-Fake-Lag";

export async function fakeLagFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const headers = new Headers(init?.headers);
  const requestedDelay = Number(headers.get(fakeLagHeader));
  const delay = Number.isFinite(requestedDelay) ? requestedDelay : 600;

  headers.delete(fakeLagHeader);
  const laggedInit = { ...init, headers };

  await new Promise((resolve) => {
    globalThis.setTimeout(resolve, delay);
  });

  return globalThis.fetch(input, laggedInit);
}
