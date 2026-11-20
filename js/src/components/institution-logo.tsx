import { LandmarkIcon } from "lucide-react";
import { useState } from "react";

import { cn } from "@/lib/utils";

interface InstitutionLogoProps {
  url: string | null;
  className?: string;
}

export function InstitutionLogo({ url, className }: InstitutionLogoProps) {
  const [failed, setFailed] = useState(false);

  if (url === null || failed) {
    return (
      <span
        className={cn(
          "flex size-8 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground",
          className,
        )}
      >
        <LandmarkIcon className="size-4" />
      </span>
    );
  }

  return (
    <img
      src={url}
      alt=""
      loading="lazy"
      className={cn("size-8 shrink-0 rounded-full", className)}
      onError={() => {
        setFailed(true);
      }}
    />
  );
}
