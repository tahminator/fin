import { Outlet, ScrollRestoration } from "react-router";

import { InstallGuide } from "@/app/_components/install-guide";
import { ThemeProvider } from "@/app/_components/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import { useIsStandalone } from "@/lib/hooks/use-is-standalone";

const installGateDisabled = import.meta.env.VITE_SKIP_INSTALL_GATE === "true";

export function RootLayout() {
  const standalone = useIsStandalone();

  return (
    <ThemeProvider>
      {standalone || installGateDisabled ? <Outlet /> : <InstallGuide />}
      <ScrollRestoration />
      <Toaster position="top-center" richColors />
    </ThemeProvider>
  );
}
