import { ErrorState } from "@/components/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { useCardArtSearch } from "@/lib/queries/card-art";
import { cn } from "@/lib/utils";

interface CardArtResultsProps {
  query: string;
  selectedCardId: string | null;
  onChoose: (cardId: string) => void;
}

export function CardArtResults({ query, selectedCardId, onChoose }: CardArtResultsProps) {
  const search = useCardArtSearch(query);

  if (query === "") {
    return <p className="text-sm text-muted-foreground">Search for your card by name.</p>;
  }

  if (search.isPending) {
    return (
      <div className="grid grid-cols-2 gap-2" aria-label="Loading card results">
        {["one", "two", "three", "four"].map((key) => (
          <div key={key} className="flex flex-col gap-1.5 rounded-lg border p-2">
            <Skeleton className="aspect-[1.586] w-full rounded-md" />
            <Skeleton className="h-3 w-3/4" />
            <Skeleton className="h-3 w-1/2" />
          </div>
        ))}
      </div>
    );
  }

  if (search.isError) {
    return <ErrorState error={search.error} onRetry={search.refetch} />;
  }

  if (search.data.length === 0) {
    return <p className="text-sm text-muted-foreground">No cards with artwork match “{query}”.</p>;
  }

  return (
    <ul className="grid max-h-80 grid-cols-2 gap-2 overflow-y-auto" aria-label="Matching cards">
      {search.data.map((card) => (
        <li key={card.id}>
          <button
            type="button"
            aria-pressed={card.id === selectedCardId}
            className={cn(
              "flex w-full flex-col gap-1.5 rounded-lg border p-2 text-left hover:bg-muted",
              card.id === selectedCardId ? "border-primary ring-2 ring-primary/30" : null,
            )}
            onClick={() => {
              onChoose(card.id);
            }}
          >
            <img
              src={card.imageUrl}
              alt=""
              loading="lazy"
              className="aspect-[1.586] w-full rounded-md object-contain"
            />
            <span className="line-clamp-2 text-xs font-medium">{card.name}</span>
            {card.issuer === null ? null : (
              <span className="text-xs text-muted-foreground">{card.issuer}</span>
            )}
          </button>
        </li>
      ))}
    </ul>
  );
}
