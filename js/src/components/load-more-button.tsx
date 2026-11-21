import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

interface LoadMoreButtonProps {
  loading: boolean;
  onLoadMore: () => Promise<unknown>;
}

export function LoadMoreButton({ loading, onLoadMore }: LoadMoreButtonProps) {
  return (
    <Button
      variant="ghost"
      className="min-h-13 w-full rounded-none font-normal"
      disabled={loading}
      onClick={() => {
        void onLoadMore();
      }}
    >
      {loading ? <Spinner /> : null}
      Load more
    </Button>
  );
}
