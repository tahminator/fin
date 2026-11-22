import { InstitutionLogo } from "@/components/institution-logo";
import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { LoadMoreButton } from "@/components/load-more-button";
import { formatDate, formatMoney } from "@/lib/format";
import { useInvestmentHoldings, useInvestmentTransactions } from "@/lib/queries/investments";

export function InvestmentResults({ accountId }: { accountId: string | undefined }) {
  const holdings = useInvestmentHoldings(accountId);
  const transactions = useInvestmentTransactions(accountId);

  if (holdings.isPending || transactions.isPending) {
    return null;
  }

  const holdingsData = holdings.data ?? [];
  const transactionsData = transactions.data ?? [];

  if (holdingsData.length === 0 && transactionsData.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-6">
      {holdingsData.length > 0 ? (
        <ListSection header="Holdings">
          {holdingsData.map((holding) => (
            <ListRow key={holding.id}>
              <div className="size-10" />
              <ListRowLabel
                title={holding.securityName ?? holding.ticker ?? "Unknown security"}
                subtitle={`${holding.quantity} shares · ${holding.accountName}`}
              />
              <p className="shrink-0 type-body tabular-nums">
                {formatMoney(holding.institutionValue, holding.currency)}
              </p>
            </ListRow>
          ))}
        </ListSection>
      ) : null}
      {transactionsData.length > 0 ? (
        <ListSection header="Investment activity">
          {transactionsData.map((transaction) => (
            <ListRow key={transaction.id}>
              <InstitutionLogo url={null} />
              <ListRowLabel
                title={
                  transaction.ticker !== null && transaction.ticker !== ""
                    ? `${transaction.ticker} · ${transaction.name}`
                    : transaction.name
                }
                subtitle={`${formatDate(transaction.postedOn)} · ${transaction.accountName}`}
              />
              <p className="shrink-0 type-body tabular-nums">
                {formatMoney(transaction.amount, transaction.currency)}
              </p>
            </ListRow>
          ))}
          {transactions.hasNextPage ? (
            <ListRow className="p-0">
              <LoadMoreButton
                loading={transactions.isFetchingNextPage}
                onLoadMore={transactions.fetchNextPage}
              />
            </ListRow>
          ) : null}
        </ListSection>
      ) : null}
    </div>
  );
}
