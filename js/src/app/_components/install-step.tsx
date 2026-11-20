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
        <CardContent>
          <CardDescription className="text-base">{step.description}</CardDescription>
        </CardContent>
      </Card>
    </div>
  );
}
