import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
          <span className="text-sm font-medium text-muted-foreground">Step {index + 1}</span>
          <CardTitle className="text-xl">{step.title}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <CardDescription className="text-base">{step.description}</CardDescription>
          {step.screenshot && (
            <div className="overflow-hidden rounded-lg border">
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
