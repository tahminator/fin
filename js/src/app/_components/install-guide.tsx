import { InstallStep } from "@/app/_components/install-step";
import { useStepScroller } from "@/app/_hooks/use-step-scroller";
import { Button } from "@/components/ui/button";
import { detectPlatform, installSteps } from "@/lib/pwa/platform";
import { cn } from "@/lib/utils";

const steps = installSteps(detectPlatform(navigator.userAgent, navigator.maxTouchPoints));

export function InstallGuide() {
  const { trackRef, step, onScroll, goTo, isFirst, isLast } = useStepScroller(steps.length);

  return (
    <main className="flex min-h-dvh flex-col justify-center gap-8 py-10">
      <header className="px-6">
        <h1 className="type-large-title font-bold">Install fin</h1>
        <p className="mt-1 type-body text-muted-foreground">
          Swipe through the steps to add fin to your device.
        </p>
      </header>

      <div
        ref={trackRef}
        onScroll={onScroll}
        className="flex snap-x snap-mandatory [scrollbar-width:none] overflow-x-auto py-1"
      >
        {steps.map((installStep, index) => (
          <InstallStep key={installStep.title} index={index} step={installStep} />
        ))}
      </div>

      <nav className="flex items-center justify-between px-6" aria-label="Install steps">
        <Button
          variant="outline"
          disabled={isFirst}
          onClick={() => {
            goTo(step - 1);
          }}
        >
          Back
        </Button>

        <div className="flex gap-2">
          {steps.map((installStep, index) => (
            <button
              key={installStep.title}
              type="button"
              aria-label={`Go to step ${index + 1}`}
              aria-current={index === step ? "step" : undefined}
              className={cn(
                "size-2.5 rounded-full bg-muted-foreground/30 transition-colors",
                index === step && "bg-primary",
              )}
              onClick={() => {
                goTo(index);
              }}
            />
          ))}
        </div>

        <Button
          disabled={isLast}
          onClick={() => {
            goTo(step + 1);
          }}
        >
          Next
        </Button>
      </nav>
    </main>
  );
}
