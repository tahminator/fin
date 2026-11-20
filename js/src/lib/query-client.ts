import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";

import { problemStatus } from "@/lib/errors";

const maxRetries = 2;

function shouldRetry(failureCount: number, error: unknown): boolean {
  const status = problemStatus(error);
  const clientError = status !== null && status >= 400 && status < 500;

  return !clientError && failureCount < maxRetries;
}

export function createQueryClient(onUnauthorized: () => void): QueryClient {
  const handleError = (error: unknown) => {
    if (problemStatus(error) === 401) {
      onUnauthorized();
    }
  };

  return new QueryClient({
    queryCache: new QueryCache({ onError: handleError }),
    mutationCache: new MutationCache({ onError: handleError }),
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        retry: shouldRetry,
      },
      mutations: {
        retry: false,
      },
    },
  });
}
