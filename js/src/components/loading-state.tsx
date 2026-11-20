import { Spinner } from "@/components/ui/spinner";

export function LoadingState() {
  return (
    <div className="flex flex-1 items-center justify-center py-16 text-muted-foreground">
      <Spinner className="size-6" />
    </div>
  );
}
