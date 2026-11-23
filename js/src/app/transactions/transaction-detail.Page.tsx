import { useParams } from "react-router";

import { TransactionDetailView } from "@/app/transactions/_components/transaction-detail";
import { DetailPanel } from "@/components/detail-panel";
import { ErrorState } from "@/components/error-state";
import { LoadingState } from "@/components/loading-state";
import { useDetailTitle } from "@/lib/hooks/use-detail-title";
import { useTransaction } from "@/lib/queries/transactions";

export function TransactionDetailPage() {
  const params = useParams();
  const { transactionId = "" } = params;
  const title = useDetailTitle("Transaction");
  const transaction = useTransaction(transactionId);

  if (transaction.isPending) {
    return (
      <DetailPanel title={title} backLabel="Transactions" backTo="/transactions">
        <LoadingState />
      </DetailPanel>
    );
  }

  if (transaction.isError) {
    return (
      <DetailPanel title={title} backLabel="Transactions" backTo="/transactions">
        <ErrorState error={transaction.error} onRetry={transaction.refetch} />
      </DetailPanel>
    );
  }

  const { data } = transaction;

  return (
    <DetailPanel title={title} backLabel="Transactions" backTo="/transactions">
      <TransactionDetailView data={data} />
    </DetailPanel>
  );
}
