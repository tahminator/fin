import { cn } from "cn";
import * as React from "react";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "h-9 w-full min-w-0 rounded-xl bg-input px-3 type-body transition-shadow outline-none file:inline-flex file:h-6 file:border-0 file:bg-transparent file:type-subheadline file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:ring-4 focus-visible:ring-ring/35 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-40 aria-invalid:ring-4 aria-invalid:ring-destructive/30",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
