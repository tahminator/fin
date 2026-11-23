import type { RouteObject } from "react-router";

import { TransactionDetailPage } from "@/app/transactions/transaction-detail.Page";
import { TransactionsPage } from "@/app/transactions/transactions.Page";

export const transactionsRoute: RouteObject = {
  path: "/transactions",
  Component: TransactionsPage,
  children: [
    {
      path: ":transactionId",
      Component: TransactionDetailPage,
    },
  ],
};
