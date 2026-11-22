import { useSearchParams } from "react-router";
import { z } from "zod";

import type { InvestmentRange } from "@/lib/types/api";

export const investmentRanges = [
  "1d",
  "3d",
  "1w",
  "1m",
  "3m",
  "1y",
  "all",
] as const satisfies readonly InvestmentRange[];

const investmentsSearchSchema = z.object({
  range: z.enum(investmentRanges).default("1m"),
  off: z.array(z.uuid()).default([]),
});

export function useInvestmentsSearch() {
  const [searchParams, setSearchParams] = useSearchParams();

  const parsed = investmentsSearchSchema.safeParse({
    range: searchParams.get("range") ?? undefined,
    off: searchParams.getAll("off"),
  });

  const update = (range: InvestmentRange, off: string[]) => {
    const params = new URLSearchParams({ range });

    for (const securityId of off) {
      params.append("off", securityId);
    }

    setSearchParams(params, { replace: true });
  };

  return { search: parsed.success ? parsed.data : null, update };
}
