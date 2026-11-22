import type { ReactNode } from "react";

interface ConsentFrameProps {
  title: string;
  subtitle?: ReactNode;
  children?: ReactNode;
}

export function ConsentFrame({ title, subtitle, children }: ConsentFrameProps) {
  return (
    <main className="mx-auto flex min-h-dvh max-w-md flex-col justify-center gap-8 px-6 pt-[env(safe-area-inset-top)] pb-[calc(env(safe-area-inset-bottom)+1.5rem)]">
      <div className="flex flex-col items-center gap-4 text-center">
        <img
          src="/logo.svg"
          alt=""
          width={64}
          height={64}
          className="size-16 rounded-[22%] shadow-[0_8px_24px_rgba(0,0,0,0.12)]"
        />
        <div className="flex flex-col gap-1">
          <h1 className="type-title2 font-bold">{title}</h1>
          {subtitle === undefined ? null : (
            <p className="type-body text-muted-foreground">{subtitle}</p>
          )}
        </div>
      </div>
      {children === undefined ? null : <div className="flex flex-col gap-4">{children}</div>}
    </main>
  );
}

export function ConsentMessage({ title, description }: { title: string; description: string }) {
  return <ConsentFrame title={title} subtitle={description} />;
}
