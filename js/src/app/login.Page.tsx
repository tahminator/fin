import { useSearchParams } from "react-router";
import { z } from "zod";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";

const loginSearchSchema = z.object({
  error: z.enum(["not_allowed", "failed"]).optional(),
});

type LoginError = NonNullable<z.infer<typeof loginSearchSchema>["error"]>;

const loginErrorMessages: Record<LoginError, string> = {
  not_allowed: "This Google account is not on the allow-list.",
  failed: "Sign-in failed. Please try again.",
};

export function LoginPage() {
  const [searchParams] = useSearchParams();
  const { error } = loginSearchSchema.parse({ error: searchParams.get("error") ?? undefined });

  return (
    <main className="mx-auto flex min-h-dvh max-w-sm flex-col justify-center gap-10 px-6 pt-[env(safe-area-inset-top)] pb-[calc(env(safe-area-inset-bottom)+1.5rem)]">
      <div className="flex flex-col items-center gap-4 text-center">
        <img
          src="/logo.svg"
          alt=""
          width={96}
          height={96}
          className="size-24 rounded-[22%] shadow-[0_8px_24px_rgba(0,0,0,0.12)]"
        />
        <div className="flex flex-col gap-1">
          <h1 className="type-large-title font-bold">fin</h1>
          <p className="type-body text-muted-foreground">
            Your accounts and transactions in one place.
          </p>
        </div>
      </div>

      <div className="flex flex-col gap-4">
        {error === undefined ? null : (
          <Alert variant="destructive">
            <AlertTitle>Can’t sign in</AlertTitle>
            <AlertDescription>{loginErrorMessages[error]}</AlertDescription>
          </Alert>
        )}

        <Button size="lg" asChild>
          <a href="/auth/google/login">Sign in with Google</a>
        </Button>
      </div>
    </main>
  );
}
