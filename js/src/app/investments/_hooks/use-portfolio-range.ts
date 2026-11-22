import { useSearchParams } from "react-router";

import { parseRange } from "@/app/investments/_components/investments-search";
import type { PortfolioRange } from "@/lib/types/api";

export function usePortfolioRange(): PortfolioRange {
  const [searchParams] = useSearchParams();

  return parseRange(searchParams.get("range"));
}
