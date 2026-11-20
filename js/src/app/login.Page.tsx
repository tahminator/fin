import { useSearchParams } from "react-router";
import { z } from "zod";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

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
    <main className="flex min-h-dvh items-center justify-center p-6">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-2xl">fin</CardTitle>
          <CardDescription>Your accounts and transactions in one place.</CardDescription>
        </CardHeader>

        <CardContent className="flex flex-col gap-4">
          {error === undefined ? null : (
            <Alert variant="destructive">
              <AlertTitle>Can’t sign in</AlertTitle>
              <AlertDescription>{loginErrorMessages[error]}</AlertDescription>
            </Alert>
          )}

          <Button size="lg" asChild>
            <a href="/auth/google/login">Sign in with Google</a>
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
