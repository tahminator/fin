import { cva } from "class-variance-authority";
import type { VariantProps } from "class-variance-authority";
import { cn } from "cn";
import { Slot } from "radix-ui";
import * as React from "react";

const buttonVariants = cva(
  "group/button inline-flex shrink-0 items-center justify-center font-semibold whitespace-nowrap transition-[opacity,transform,background-color] duration-150 outline-none select-none focus-visible:ring-4 focus-visible:ring-ring/35 active:not-disabled:scale-[0.97] active:not-disabled:opacity-70 disabled:pointer-events-none disabled:opacity-40 aria-invalid:ring-4 aria-invalid:ring-destructive/30 motion-reduce:transition-none motion-reduce:active:not-disabled:scale-100 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground",
        outline: "bg-secondary text-primary aria-expanded:bg-fill",
        secondary: "bg-primary/15 text-primary aria-expanded:bg-primary/25",
        ghost: "text-primary hover:bg-accent aria-expanded:bg-accent",
        destructive: "bg-destructive/15 text-destructive focus-visible:ring-destructive/30",
        link: "text-primary",
      },
      size: {
        default: "h-9 gap-1.5 rounded-xl px-4 type-body",
        xs: "h-6 gap-1 rounded-lg px-2 type-caption1 [&_svg:not([class*='size-'])]:size-3",
        sm: "h-7 gap-1 rounded-[10px] px-3 type-subheadline [&_svg:not([class*='size-'])]:size-3.5",
        lg: "h-[50px] gap-2 rounded-[14px] px-5 type-body",
        icon: "size-9 rounded-full [&_svg:not([class*='size-'])]:size-5",
        "icon-xs": "size-6 rounded-full [&_svg:not([class*='size-'])]:size-3",
        "icon-sm": "size-7 rounded-full",
        "icon-lg": "size-11 rounded-full [&_svg:not([class*='size-'])]:size-5",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

function Button({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
  }) {
  const Comp = asChild ? Slot.Root : "button";

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Button, buttonVariants };
