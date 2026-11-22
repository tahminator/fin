import { createBrowserRouter } from "react-router";

import { AuthenticatedLayout } from "@/app/_components/authenticated-layout";
import { RootLayout } from "@/app/_components/root-layout";
import { ConnectionsPage } from "@/app/connections/connections.Page";
import { OAuthReturnPage } from "@/app/connections/oauth.Page";
import { HomePage } from "@/app/home.Page";
import { InvestmentsPage } from "@/app/investments/investments.Page";
import { LoginPage } from "@/app/login.Page";
import { TransactionsPage } from "@/app/transactions/transactions.Page";
import { createQueryClient } from "@/lib/query-client";

export const queryClient = createQueryClient(() => {
  if (router.state.location.pathname === "/consent") {
    return;
  }

  queryClient.clear();

  void router.navigate("/login", { replace: true });
});

export const router = createBrowserRouter([
  {
    Component: RootLayout,
    children: [
      { path: "/login", Component: LoginPage },
      {
        path: "/consent",
        lazy: async () => {
          const { ConsentPage } = await import("@/app/consent/consent.Page");

          return { Component: ConsentPage };
        },
      },
      {
        Component: AuthenticatedLayout,
        children: [
          { index: true, Component: HomePage },
          { path: "/connections", Component: ConnectionsPage },
          { path: "/connections/oauth", Component: OAuthReturnPage },
          { path: "/transactions", Component: TransactionsPage },
          { path: "/investments", Component: InvestmentsPage },
          {
            path: "/apps",
            lazy: async () => {
              const { AppsPage } = await import("@/app/apps/apps.Page");

              return { Component: AppsPage };
            },
          },
        ],
      },
    ],
  },
]);
