import { Link, useSearchParams } from "react-router";

import {
  investmentsLink,
  investmentsSearchSchema,
} from "@/app/investments/_components/investments-search";
import { PortfolioView } from "@/app/investments/_components/portfolio-view";
import { EmptyState } from "@/components/empty-state";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";

export function InvestmentsPage() {
  const [searchParams] = useSearchParams();
  const search = investmentsSearchSchema.safeParse({
    range: searchParams.get("range") ?? undefined,
    accountId: searchParams.get("accountId") ?? undefined,
    securityId: searchParams.get("securityId") ?? undefined,
  });

  if (!search.success) {
    return (
      <>
        <PageHeader title="Investments" />
        <EmptyState
          title="Invalid investments link"
          action={
            <Button variant="outline" asChild>
              <Link to="/investments">All investments</Link>
            </Button>
          }
        />
      </>
    );
  }

  const scoped = search.data.accountId !== undefined || search.data.securityId !== undefined;

  return (
    <>
      <PageHeader
        title="Investments"
        actions={
          scoped ? (
            <Button variant="outline" size="sm" asChild>
              <Link to={investmentsLink({ range: search.data.range })}>All investments</Link>
            </Button>
          ) : undefined
        }
      />
      <PortfolioView search={search.data} />
    </>
  );
}
