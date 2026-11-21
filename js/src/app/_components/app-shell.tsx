import { HouseIcon, LandmarkIcon, ReceiptTextIcon } from "lucide-react";
import type { ReactNode } from "react";
import { NavLink, Outlet } from "react-router";

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
      className="absolute top-1 left-1/2 ml-2 h-4 min-w-4 px-1 text-[10px] tabular-nums"
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

function NavItem({ to, label, icon, badge }: NavItemProps) {
  return (
    <NavLink
      to={to}
      end={to === "/"}
      className={({ isActive }) =>
        `relative flex flex-1 flex-col items-center gap-1 py-2 text-xs ${isActive ? "font-semibold text-primary" : "text-muted-foreground"}`
      }
    >
      {icon}
      {label}
      {badge}
    </NavLink>
  );
}

export function AppShell() {
  return (
    <div className="mx-auto flex min-h-dvh max-w-2xl flex-col">
      <header className="sticky top-0 z-10 flex items-center justify-between border-b bg-background px-4 pt-[env(safe-area-inset-top)]">
        <span className="py-3 text-xl font-bold tracking-tight text-primary">fin</span>
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

      <nav className="sticky bottom-0 flex border-t bg-background pb-[env(safe-area-inset-bottom)]">
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
