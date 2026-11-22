import { ChartLineIcon, HouseIcon, LandmarkIcon, ReceiptTextIcon } from "lucide-react";
import { ViewTransition } from "react";
import type { ReactNode } from "react";
import { NavLink } from "react-router";

import { Badge } from "@/components/ui/badge";
import { useNotifications } from "@/lib/queries/notifications";
import { cn } from "@/lib/utils";

function ReauthBadge() {
  const notifications = useNotifications();

  if (!notifications.isSuccess || notifications.data.total === 0) {
    return null;
  }

  return (
    <Badge
      variant="destructive"
      className="absolute top-1 left-1/2 ml-1.5 h-4.5 min-w-4.5 bg-destructive px-1 type-caption2 text-white"
    >
      {notifications.data.total}
    </Badge>
  );
}

interface TabItemProps {
  to: "/" | "/transactions" | "/investments" | "/connections";
  label: string;
  icon: ReactNode;
  badge?: ReactNode;
}

function TabItem({ to, label, icon, badge }: TabItemProps) {
  return (
    <NavLink
      to={to}
      end={to === "/"}
      className={({ isActive }) =>
        cn(
          "relative isolate flex min-h-13.5 flex-1 flex-col items-center justify-center gap-0.5 rounded-full px-2 type-caption2 font-medium transition-[color,transform] duration-150 outline-none focus-visible:ring-4 focus-visible:ring-ring/35 active:scale-[0.94] motion-reduce:transition-none motion-reduce:active:scale-100",
          isActive ? "text-primary" : "text-foreground",
        )
      }
    >
      {({ isActive }) => (
        <>
          {isActive ? (
            <ViewTransition name="tab-indicator" share="tab-indicator">
              <span aria-hidden="true" className="absolute inset-0 -z-10 rounded-full bg-fill" />
            </ViewTransition>
          ) : null}
          {icon}
          {label}
          {badge}
        </>
      )}
    </NavLink>
  );
}

export function TabBar() {
  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-20 flex justify-center px-6 pb-[max(calc(env(safe-area-inset-bottom)-0.5rem),1rem)]">
      <nav
        aria-label="Primary"
        className="pointer-events-auto flex w-full max-w-sm gap-1 rounded-full glass p-1"
      >
        <TabItem to="/" label="Home" icon={<HouseIcon className="size-6" />} />
        <TabItem
          to="/transactions"
          label="Transactions"
          icon={<ReceiptTextIcon className="size-6" />}
        />
        <TabItem
          to="/investments"
          label="Investments"
          icon={<ChartLineIcon className="size-6" />}
        />
        <TabItem
          to="/connections"
          label="Connections"
          icon={<LandmarkIcon className="size-6" />}
          badge={<ReauthBadge />}
        />
      </nav>
    </div>
  );
}
