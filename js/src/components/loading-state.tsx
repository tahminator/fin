import { Skeleton } from "@/components/ui/skeleton";

export function LoadingState() {
  return (
    <div className="flex flex-1 flex-col gap-6 py-1" aria-label="Loading">
      <Skeleton className="h-32 w-full rounded-[26px]" />
      <div className="flex flex-col gap-1.5 rounded-[26px] border p-2">
        {["one", "two", "three"].map((key) => (
          <div key={key} className="flex min-h-15 items-center gap-3 px-2">
            <Skeleton className="size-9 rounded-full" />
            <div className="flex flex-1 flex-col gap-2">
              <Skeleton className="h-4 w-2/5" />
              <Skeleton className="h-3 w-1/3" />
            </div>
            <Skeleton className="h-4 w-16" />
          </div>
        ))}
      </div>
    </div>
  );
}
