import { InvestmentActivitySection } from "@/app/transactions/_components/investment-activity-section";
import { InvestmentHoldingsSection } from "@/app/transactions/_components/investment-holdings-section";
import { TransactionResults } from "@/app/transactions/_components/transaction-results";
import { EmptyState } from "@/components/empty-state";
import { useInvestmentHoldings, useInvestmentTransactions } from "@/lib/queries/investments";
import { useTransactions } from "@/lib/queries/transactions";

export function TransactionsContent({
  accountId,
  hide,
}: {
  accountId: string | undefined;
  hide: string[];
}) {
  const scope = { accountId, excludeAccountId: hide };
  const holdings = useInvestmentHoldings(scope);
  const activity = useInvestmentTransactions(scope);
  const transactions = useTransactions(accountId, hide);

  const hasInvestmentData = (holdings.data?.length ?? 0) > 0 || (activity.data?.length ?? 0) > 0;
  const nothingAnywhere =
    holdings.isSuccess &&
    activity.isSuccess &&
    transactions.isSuccess &&
    !hasInvestmentData &&
    transactions.data.length === 0;

  if (nothingAnywhere) {
    return hide.length > 0 ? (
      <EmptyState
        title="Nothing to show"
        description="The selected accounts have no transactions or holdings. Hidden accounts are left out."
      />
    ) : (
      <EmptyState title="No transactions yet" />
    );
  }

  return (
    <div className="flex flex-col gap-8">
      <InvestmentHoldingsSection holdings={holdings} />
      <InvestmentActivitySection activity={activity} />
      <TransactionResults transactions={transactions} showHeading={hasInvestmentData} />
    </div>
  );
}
