import { useQueryClient } from "@tanstack/react-query";

import { $api } from "@/lib/api/client";

const cardArtSearchCacheTime = 7 * 24 * 60 * 60 * 1000;

export function useCardArtSearch(query: string) {
  return $api.useQuery(
    "get",
    "/api/card-art/search",
    { params: { query: { query } } },
    {
      enabled: query !== "",
      staleTime: cardArtSearchCacheTime,
      gcTime: cardArtSearchCacheTime,
      refetchOnWindowFocus: false,
      select: (data) => data.items,
    },
  );
}

export function useSelectCardArt() {
  const queryClient = useQueryClient();

  return $api.useMutation("put", "/api/accounts/{accountId}/card-art", {
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["get", "/api/accounts"] }),
  });
}
