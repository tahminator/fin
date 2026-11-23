import { createMemoryRouter, createPath } from "react-router";
import type { Location } from "react-router";

import { AuthenticatedLayout } from "@/app/_components/authenticated-layout";
import { RootLayout } from "@/app/_components/root-layout";
import { ConnectionsPage } from "@/app/connections/connections.Page";
import { OAuthReturnPage } from "@/app/connections/oauth.Page";
import { HomePage } from "@/app/home.Page";
import { investmentsRoute } from "@/app/investments/investments-route";
import { LoginPage } from "@/app/login.Page";
import { transactionsRoute } from "@/app/transactions/transactions-route";
import { createQueryClient } from "@/lib/query-client";

export const queryClient = createQueryClient(() => {
  if (router.state.location.pathname === "/consent") {
    return;
  }

  queryClient.clear();

  void router.navigate("/login", { replace: true });
});

const { pathname, search, hash } = globalThis.location;

export const router = createMemoryRouter(
  [
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
            transactionsRoute,
            investmentsRoute,
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
  ],
  { initialEntries: [`${pathname}${search}${hash}`] },
);

const addressablePaths = new Set(["/consent", "/connections/oauth"]);

function syncAddressBar(location: Location) {
  const url = addressablePaths.has(location.pathname) ? createPath(location) : "/";

  if (url !== createPath(globalThis.location)) {
    globalThis.history.replaceState(null, "", url);
  }
}

syncAddressBar(router.state.location);

router.subscribe(({ location }) => {
  syncAddressBar(location);
});
