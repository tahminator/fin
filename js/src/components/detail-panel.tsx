import { ChevronLeftIcon } from "lucide-react";
import type { ReactNode } from "react";
import { useLocation, useNavigate } from "react-router";
import type { To } from "react-router";

import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { useSwipeBack } from "@/lib/hooks/use-swipe-back";

interface DetailPanelProps {
  title: string;
  backLabel: string;
  backTo: To;
  children: ReactNode;
}

export function DetailPanel({ title, backLabel, backTo, children }: DetailPanelProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const { panelRef, back, handlers } = useSwipeBack(() => {
    if (location.key === "default") {
      void navigate(backTo, { replace: true, preventScrollReset: true });
    } else {
      void navigate(-1);
    }
  });

  return (
    <div
      ref={panelRef}
      {...handlers}
      className="fixed inset-0 z-25 animate-push-in overflow-y-auto overscroll-contain bg-background shadow-[-12px_0_32px_rgba(0,0,0,0.18)]"
    >
      <header className="sticky top-0 z-10 pt-[env(safe-area-inset-top)]">
        <div className="mx-auto flex h-13.5 max-w-2xl items-center px-4">
          <Button
            variant="ghost"
            className="h-11 gap-0.5 rounded-full glass-sm pr-4 pl-2 text-primary"
            onClick={back}
          >
            <ChevronLeftIcon className="size-6" />
            <span className="max-w-48 truncate">{backLabel}</span>
          </Button>
        </div>
      </header>
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-6 px-4 pt-1 pb-[calc(env(safe-area-inset-bottom)+6.5rem)]">
        <PageHeader title={title} />
        {children}
      </div>
    </div>
  );
}
