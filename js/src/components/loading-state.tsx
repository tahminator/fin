import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

const placeholderRows = ["first", "second", "third", "fourth", "fifth"];

export function LoadingState({ className }: { className?: string }) {
  return (
    <output aria-label="Loading" className={cn("flex flex-col divide-y", className)}>
      {placeholderRows.map((row) => (
        <div key={row} className="flex items-center gap-3 py-3">
          <Skeleton className="size-8 shrink-0 rounded-full" />
          <div className="flex flex-1 flex-col gap-2">
            <Skeleton className="h-4 w-2/5" />
            <Skeleton className="h-3 w-1/4" />
          </div>
          <Skeleton className="h-4 w-16" />
        </div>
      ))}
    </output>
  );
}
