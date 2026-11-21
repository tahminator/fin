import { HouseIcon, LandmarkIcon, ReceiptTextIcon } from "lucide-react";
import type { ReactNode } from "react";
import { NavLink, Outlet, useLocation } from "react-router";

import { ReauthBanner } from "@/app/_components/reauth-banner";
import { RefreshButton } from "@/app/_components/refresh-button";
import { SignOutButton } from "@/app/_components/sign-out-button";
import { ThemeToggle } from "@/app/_components/theme-provider";
import { Badge } from "@/components/ui/badge";
import { useNotifications } from "@/lib/queries/notifications";

function ReauthBadge() {
  const notifications = useNotifications();

  if (!notifications.isSuccess || notifications.data.total === 0) {
    return null;
  }

  return (
    <Badge
      variant="destructive"
      className="absolute top-1 left-1/2 ml-2 h-4 min-w-4 px-1 text-[10px]"
    >
      {notifications.data.total}
    </Badge>
  );
}

interface NavItemProps {
  to: "/" | "/transactions" | "/connections";
  label: string;
  icon: ReactNode;
  badge?: ReactNode;
}

function getActiveIndex(pathname: string) {
  if (pathname === "/") {
    return 0;
  }
  if (pathname.startsWith("/transactions")) {
    return 1;
  }
  return 2;
}

function NavItem({ to, label, icon, badge }: NavItemProps) {
  return (
    <NavLink
      to={to}
      end={to === "/"}
      className={({ isActive }) =>
        `relative z-10 flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 text-xs transition-colors duration-300 ${isActive ? "text-foreground" : "text-muted-foreground"}`
      }
    >
      {icon}
      {label}
      {badge}
    </NavLink>
  );
}

export function AppShell() {
  const { pathname } = useLocation();
  const activeIndex = getActiveIndex(pathname);

  return (
    <div className="mx-auto flex min-h-dvh max-w-2xl flex-col">
      <header className="sticky top-0 z-10 flex items-center justify-between border-b bg-background/95 px-4 pt-[env(safe-area-inset-top)] backdrop-blur">
        <span className="py-3 text-lg font-semibold tracking-tight">fin</span>
        <div className="flex items-center gap-1">
          <ThemeToggle />
          <RefreshButton />
          <SignOutButton />
        </div>
      </header>

      <main className="flex flex-1 flex-col gap-4 p-4">
        <ReauthBanner />
        <Outlet />
      </main>

      <nav
        aria-label="Primary navigation"
        className="sticky bottom-0 z-20 mx-3 mb-[calc(env(safe-area-inset-bottom)+12px)] flex rounded-full border border-border/70 bg-background/75 p-1.5 shadow-lg shadow-black/5 backdrop-blur-xl"
      >
        <span
          aria-hidden="true"
          className="absolute top-1.5 bottom-1.5 left-1.5 rounded-full bg-accent shadow-sm transition-transform duration-300 ease-out"
          style={{
            width: "calc((100% - 12px) / 3)",
            transform: `translateX(${activeIndex * 100}%)`,
          }}
        />
        <NavItem to="/" label="Home" icon={<HouseIcon className="size-5" />} />
        <NavItem
          to="/transactions"
          label="Transactions"
          icon={<ReceiptTextIcon className="size-5" />}
        />
        <NavItem
          to="/connections"
          label="Connections"
          icon={<LandmarkIcon className="size-5" />}
          badge={<ReauthBadge />}
        />
      </nav>
    </div>
  );
}
