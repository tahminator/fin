import { ShieldAlertIcon, ShieldCheckIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import type { ConsentRequest } from "@/lib/types/api";

export function VerificationBadge({ authorization }: { authorization: ConsentRequest }) {
  if (authorization.verified) {
    return (
      <Badge variant="secondary" className="gap-1">
        <ShieldCheckIcon className="size-3.5" />
        Verified: {authorization.clientHost}
      </Badge>
    );
  }

  return (
    <Badge variant="destructive" className="gap-1">
      <ShieldAlertIcon className="size-3.5" />
      Unverified app
    </Badge>
  );
}
