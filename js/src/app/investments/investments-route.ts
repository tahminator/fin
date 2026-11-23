import type { RouteObject } from "react-router";

import { InvestmentAccountPage } from "@/app/investments/accounts/account.Page";
import { InvestmentHoldingPage } from "@/app/investments/holdings/holding.Page";
import { InvestmentsPage } from "@/app/investments/investments.Page";
import { InvestmentTransactionDetailPage } from "@/app/investments/transactions/investment-transaction-detail.Page";

export const investmentsRoute: RouteObject = {
  path: "/investments",
  Component: InvestmentsPage,
  children: [
    {
      path: "transactions/:transactionId",
      Component: InvestmentTransactionDetailPage,
    },
    {
      path: "accounts/:accountId",
      Component: InvestmentAccountPage,
      children: [{ path: "holdings/:securityId", Component: InvestmentHoldingPage }],
    },
    { path: "holdings/:securityId", Component: InvestmentHoldingPage },
  ],
};
