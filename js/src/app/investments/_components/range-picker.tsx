import { Link } from "react-router";

import type { InvestmentsSearch } from "@/app/investments/_components/investments-search";
import { investmentsLink, portfolioRanges } from "@/app/investments/_components/investments-search";
import { cn } from "@/lib/utils";

export function RangePicker({ search }: { search: InvestmentsSearch }) {
  return (
    <nav aria-label="Time range" className="flex gap-1">
      {portfolioRanges.map((option) => {
        const active = option.value === search.range;

        return (
          <Link
            key={option.value}
            to={investmentsLink({ ...search, range: option.value })}
            replace
            preventScrollReset
            aria-current={active ? "page" : undefined}
            aria-label={option.description}
            className={cn(
              "flex h-8 flex-1 items-center justify-center rounded-full type-footnote font-semibold transition-colors outline-none focus-visible:ring-4 focus-visible:ring-ring/35",
              active ? "bg-fill text-foreground" : "text-muted-foreground active:bg-fill",
            )}
          >
            {option.label}
          </Link>
        );
      })}
    </nav>
  );
}
