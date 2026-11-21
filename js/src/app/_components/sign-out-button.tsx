import { LogOutIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { errorMessage } from "@/lib/errors";
import { useLogout } from "@/lib/queries/me";

export function SignOutButton() {
  const logout = useLogout();

  return (
    <Button
      variant="ghost"
      size="icon-lg"
      className="text-foreground"
      aria-label="Sign out"
      disabled={logout.isPending}
      onClick={() => {
        logout.mutate(
          {},
          {
            onError: (error) => {
              toast.error("Sign out failed", { description: errorMessage(error) });
            },
          },
        );
      }}
    >
      <LogOutIcon />
    </Button>
  );
}
