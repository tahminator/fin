import { ConsentFrame } from "@/app/consent/_components/consent-frame";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";

export type LoginError = "not_allowed" | "failed";

const loginErrorMessages: Record<LoginError, string> = {
  not_allowed: "This Google account is not on the allow-list, so it can’t connect apps to fin.",
  failed: "Sign-in failed. Please try again.",
};

interface SignInToContinueProps {
  request: string;
  error: LoginError | null;
}

export function SignInToContinue({ request, error }: SignInToContinueProps) {
  return (
    <ConsentFrame
      title="Sign in to continue"
      subtitle="An app wants to connect to your fin account."
    >
      {error === null ? null : (
        <Alert variant="destructive">
          <AlertTitle>Can’t sign in</AlertTitle>
          <AlertDescription>{loginErrorMessages[error]}</AlertDescription>
        </Alert>
      )}
      <Button size="lg" asChild>
        <a href={`/auth/google/login?consent=${encodeURIComponent(request)}`}>
          Sign in with Google
        </a>
      </Button>
    </ConsentFrame>
  );
}
