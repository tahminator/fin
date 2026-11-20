import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";

import { $api } from "@/lib/api/client";

export function meQueryOptions() {
  return $api.queryOptions("get", "/api/me");
}

export function useMe() {
  return useQuery({ ...meQueryOptions(), staleTime: "static" });
}

export function useLogout() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return $api.useMutation("post", "/api/auth/logout", {
    onSuccess: async () => {
      queryClient.clear();

      await navigate("/login", { replace: true });
    },
  });
}
