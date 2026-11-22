import { Link } from "react-router";

import { EmptyState } from "@/components/empty-state";
import { Button } from "@/components/ui/button";

export function EmptyPortfolio() {
  return (
    <EmptyState
      title="No investments yet"
      description="Connect a brokerage or retirement account to track your portfolio here."
      action={
        <Button asChild>
          <Link to="/connections">Connect an account</Link>
        </Button>
      }
    />
  );
}
