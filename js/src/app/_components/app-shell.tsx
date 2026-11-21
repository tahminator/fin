import { Outlet } from "react-router";

import { ReauthBanner } from "@/app/_components/reauth-banner";
import { TabBar } from "@/app/_components/tab-bar";
import { Toolbar } from "@/app/_components/toolbar";

export function AppShell() {
  return (
    <div className="flex min-h-dvh flex-col">
      <Toolbar />
      <main className="mx-auto flex w-full max-w-2xl flex-1 flex-col gap-6 px-4 pt-1 pb-[calc(env(safe-area-inset-bottom)+6.5rem)]">
        <ReauthBanner />
        <Outlet />
      </main>
      <TabBar />
    </div>
  );
}
