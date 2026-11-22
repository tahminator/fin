import { Outlet, ScrollRestoration, useMatch } from "react-router";

import { InstallGuide } from "@/app/_components/install-guide";
import { ThemeProvider } from "@/app/_components/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import { useIsStandalone } from "@/lib/hooks/use-is-standalone";
import { detectPlatform, isDesktopPlatform } from "@/lib/pwa/platform";

const installGateDisabled = import.meta.env.VITE_SKIP_INSTALL_GATE === "true";
const platform = detectPlatform(navigator.userAgent, navigator.maxTouchPoints);
const desktopBrowser = isDesktopPlatform(platform);

export function RootLayout() {
  const standalone = useIsStandalone();
  const consent = useMatch("/consent") !== null;

  return (
    <ThemeProvider>
      {standalone || consent || desktopBrowser || installGateDisabled ? (
        <Outlet />
      ) : (
        <InstallGuide />
      )}
      <ScrollRestoration />
      <Toaster position="top-center" richColors />
    </ThemeProvider>
  );
}
