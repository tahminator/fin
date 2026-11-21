import { Card, CardContent, CardHeader } from "@/components/ui/card";
import type { InstallStep as InstallStepData } from "@/lib/pwa/platform";

interface InstallStepProps {
  index: number;
  step: InstallStepData;
}

export function InstallStep({ index, step }: InstallStepProps) {
  return (
    <div className="w-full shrink-0 snap-center px-6">
      <Card className="h-full">
        <CardHeader>
          <span className="type-subheadline font-semibold text-primary">Step {index + 1}</span>
          <h2 className="type-title2 font-bold">{step.title}</h2>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <p className="type-body text-muted-foreground">{step.description}</p>
          {step.screenshot && (
            <div className="overflow-hidden rounded-2xl ring-[0.5px] ring-border">
              <img
                src={step.screenshot.light}
                alt={step.screenshot.alt}
                width={720}
                height={645}
                loading="lazy"
                className="h-auto w-full dark:hidden"
              />
              <img
                src={step.screenshot.dark}
                alt={step.screenshot.alt}
                width={720}
                height={645}
                loading="lazy"
                className="hidden h-auto w-full dark:block"
              />
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
