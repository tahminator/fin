import { RefreshButton } from "@/app/_components/refresh-button";
import { SignOutButton } from "@/app/_components/sign-out-button";
import { ThemeToggle } from "@/app/_components/theme-provider";
import { useIsScrolled } from "@/lib/hooks/use-is-scrolled";
import { cn } from "@/lib/utils";

export function Toolbar() {
  const scrolled = useIsScrolled();

  return (
    <header
      className={cn(
        "sticky top-0 z-20 border-b border-transparent pt-[env(safe-area-inset-top)] transition-[background-color,border-color] duration-250",
        scrolled ? "toolbar-uniform" : null,
      )}
    >
      <div className="mx-auto flex h-13.5 max-w-2xl items-center justify-end px-4">
        <div className="flex items-center rounded-full glass-sm">
          <ThemeToggle />
          <RefreshButton />
          <SignOutButton />
        </div>
      </div>
    </header>
  );
}
