import type { ComponentProps, ReactNode } from "react";

import { cn } from "@/lib/utils";

interface ListSectionProps {
  header?: ReactNode;
  footer?: ReactNode;
  label?: string;
  children: ReactNode;
}

export function ListSection({ header, footer, label, children }: ListSectionProps) {
  return (
    <section className="flex flex-col gap-1.5">
      {header === undefined ? null : (
        <h2 className="px-4 type-footnote text-muted-foreground uppercase">{header}</h2>
      )}
      <ul aria-label={label} className="overflow-hidden rounded-[26px] bg-card">
        {children}
      </ul>
      {footer === undefined ? null : (
        <div className="px-4 type-footnote text-muted-foreground">{footer}</div>
      )}
    </section>
  );
}

export function ListRow({ className, ...props }: ComponentProps<"li">) {
  return (
    <li
      className={cn(
        "relative flex min-h-13 items-center gap-3 px-4 py-2 not-first:before:absolute not-first:before:top-0 not-first:before:right-0 not-first:before:left-15 not-first:before:h-px not-first:before:scale-y-50 not-first:before:bg-border",
        className,
      )}
      {...props}
    />
  );
}

interface ListRowLabelProps {
  title: ReactNode;
  subtitle?: ReactNode;
}

export function ListRowLabel({ title, subtitle }: ListRowLabelProps) {
  return (
    <div className="min-w-0 flex-1">
      <div className="truncate type-body">{title}</div>
      {subtitle === undefined ? null : (
        <div className="truncate type-subheadline text-muted-foreground">{subtitle}</div>
      )}
    </div>
  );
}
