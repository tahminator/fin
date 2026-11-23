import { useParams } from "react-router";

import { InvestmentTransactionDetail } from "@/app/investments/_components/investment-transaction-detail";
import { portfolioLink } from "@/app/investments/_components/investments-search";
import { usePortfolioRange } from "@/app/investments/_hooks/use-portfolio-range";
import { DetailPanel } from "@/components/detail-panel";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { useDetailTitle } from "@/lib/hooks/use-detail-title";
import { useInvestmentTransaction } from "@/lib/queries/investments";

export function InvestmentTransactionDetailPage() {
  const params = useParams();
  const { transactionId = "" } = params;
  const range = usePortfolioRange();
  const title = useDetailTitle("Investment activity");
  const transaction = useInvestmentTransaction(transactionId);

  if (transaction.isPending) {
    return (
      <DetailPanel title={title} backLabel="Investments" backTo={portfolioLink(range)}>
        <LoadingState />
      </DetailPanel>
    );
  }

  if (transaction.isError) {
    return (
      <DetailPanel title={title} backLabel="Investments" backTo={portfolioLink(range)}>
        <ErrorState error={transaction.error} onRetry={transaction.refetch} />
      </DetailPanel>
    );
  }

  const { data } = transaction;

  return (
    <DetailPanel title={title} backLabel="Investments" backTo={portfolioLink(range)}>
      <InvestmentTransactionDetail data={data} />
    </DetailPanel>
  );
}
