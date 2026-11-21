import type { ReactNode } from "react";

interface PageHeaderProps {
  title: string;
  actions?: ReactNode;
}

export function PageHeader({ title, actions }: PageHeaderProps) {
  return (
    <header className="order-first flex flex-wrap items-end justify-between gap-x-3 gap-y-2 px-1">
      <h1 className="type-large-title font-bold">{title}</h1>
      {actions === undefined ? null : (
        <div className="flex shrink-0 items-center gap-2 pb-1">{actions}</div>
      )}
    </header>
  );
}
