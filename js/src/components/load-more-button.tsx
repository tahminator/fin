import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

interface LoadMoreButtonProps {
  loading: boolean;
  onLoadMore: () => Promise<unknown>;
}

export function LoadMoreButton({ loading, onLoadMore }: LoadMoreButtonProps) {
  return (
    <Button
      variant="outline"
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
