import { useLocation } from "react-router";
import { z } from "zod";

const titleState = z.object({ title: z.string() });

export function useDetailTitle(fallback: string): string {
  const location = useLocation();
  const state = titleState.safeParse(location.state);

  return state.success ? state.data.title : fallback;
}
