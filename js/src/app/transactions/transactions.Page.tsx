import { Link, useSearchParams } from "react-router";
import { z } from "zod";

import { TransactionResults } from "@/app/transactions/_components/transaction-results";
import { TransactionsFilter } from "@/app/transactions/_components/transactions-filter";
import { EmptyState } from "@/components/empty-state";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";

const transactionsSearchSchema = z.object({
  accountId: z.uuid().optional(),
  hide: z.array(z.uuid()).default([]),
});

function hideSearchParams(hide: string[]): URLSearchParams {
  const params = new URLSearchParams();

  for (const accountId of hide) {
    params.append("hide", accountId);
  }

  return params;
}

export function TransactionsPage() {
  const [searchParams] = useSearchParams();
  const search = transactionsSearchSchema.safeParse({
    accountId: searchParams.get("accountId") ?? undefined,
    hide: searchParams.getAll("hide"),
  });

  if (!search.success) {
    return (
      <>
        <PageHeader title="Transactions" />
        <EmptyState
          title="Invalid account filter"
          action={
            <Button variant="outline" asChild>
              <Link to="/transactions">All accounts</Link>
            </Button>
          }
        />
      </>
    );
  }

  const { accountId, hide } = search.data;

  return (
    <>
      <PageHeader
        title="Transactions"
        actions={
          accountId === undefined ? (
            <TransactionsFilter />
          ) : (
            <Button variant="outline" size="sm" asChild>
              <Link to={{ pathname: "/transactions", search: hideSearchParams(hide).toString() }}>
                All accounts
              </Link>
            </Button>
          )
        }
      />
      <TransactionResults accountId={accountId} excludeAccountId={hide} />
    </>
  );
}
