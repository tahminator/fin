import iosAddDark from "@/assets/install/ios-add-dark.webp";
import iosAddLight from "@/assets/install/ios-add-light.webp";
import iosAddToHomeScreenDark from "@/assets/install/ios-add-to-home-screen-dark.webp";
import iosAddToHomeScreenLight from "@/assets/install/ios-add-to-home-screen-light.webp";
import iosPageMenuDark from "@/assets/install/ios-page-menu-dark.webp";
import iosPageMenuLight from "@/assets/install/ios-page-menu-light.webp";
import iosShareDark from "@/assets/install/ios-share-dark.webp";
import iosShareLight from "@/assets/install/ios-share-light.webp";
import iosViewMoreDark from "@/assets/install/ios-view-more-dark.webp";
import iosViewMoreLight from "@/assets/install/ios-view-more-light.webp";

export type Platform =
  | "ios-safari"
  | "ios-other"
  | "android-chrome"
  | "desktop-chromium"
  | "desktop-safari";

interface InstallStepScreenshot {
  light: string;
  dark: string;
  alt: string;
}

export interface InstallStep {
  title: string;
  description: string;
  screenshot?: InstallStepScreenshot;
}

const iosSafariSteps: InstallStep[] = [
  {
    title: "Tap the page menu button",
    description: "It's next to the address bar at the bottom of Safari.",
    screenshot: {
      light: iosPageMenuLight,
      dark: iosPageMenuDark,
      alt: "Safari toolbar with the page menu button highlighted next to the address bar",
    },
  },
  {
    title: "Tap Share",
    description: "It's at the top of the menu.",
    screenshot: {
      light: iosShareLight,
      dark: iosShareDark,
      alt: "Safari page menu with Share highlighted",
    },
  },
  {
    title: "Tap View More",
    description: "Skip this if Add to Home Screen is already showing.",
    screenshot: {
      light: iosViewMoreLight,
      dark: iosViewMoreDark,
      alt: "Share sheet with View More highlighted",
    },
  },
  {
    title: "Tap Add to Home Screen",
    description: "It's in the list of actions below the share options.",
    screenshot: {
      light: iosAddToHomeScreenLight,
      dark: iosAddToHomeScreenDark,
      alt: "Expanded share sheet with Add to Home Screen highlighted",
    },
  },
  {
    title: "Tap Add, then open fin",
    description: "Leave Open as Web App on. Launch fin from your Home Screen and sign in there.",
    screenshot: {
      light: iosAddLight,
      dark: iosAddDark,
      alt: "Add to Home Screen sheet with the Add button and Open as Web App switch highlighted",
    },
  },
];

const stepsByPlatform: Record<Platform, InstallStep[]> = {
  "ios-safari": iosSafariSteps,
  "ios-other": [
    {
      title: "Open this page in Safari",
      description: "The steps that follow are for Safari.",
    },
    ...iosSafariSteps,
  ],
  "android-chrome": [
    {
      title: "Tap ⋮",
      description: "It's the menu button in the top-right corner of Chrome.",
    },
    {
      title: "Tap Install app",
      description: "Confirm the prompt to add fin to your home screen.",
    },
    {
      title: "Open fin from your home screen",
      description: "Sign in from the installed app.",
    },
  ],
  "desktop-chromium": [
    {
      title: "Click the install icon in the address bar",
      description: "It's on the right side of the address bar, then confirm Install.",
    },
    {
      title: "Open fin from your apps",
      description: "fin opens in its own window. Sign in there.",
    },
  ],
  "desktop-safari": [
    {
      title: "Choose File → Add to Dock",
      description: "Confirm the name and click Add.",
    },
    {
      title: "Open fin from the Dock",
      description: "fin opens in its own window. Sign in there.",
    },
  ],
};

const iosOtherBrowserPattern = /CriOS|FxiOS|EdgiOS|OPiOS|GSA\//u;

export function detectPlatform(userAgent: string, maxTouchPoints: number): Platform {
  const iPadOnDesktopMode = userAgent.includes("Macintosh") && maxTouchPoints > 1;

  if (/iPhone|iPad|iPod/u.test(userAgent) || iPadOnDesktopMode) {
    return iosOtherBrowserPattern.test(userAgent) ? "ios-other" : "ios-safari";
  }

  if (userAgent.includes("Android")) {
    return "android-chrome";
  }

  if (userAgent.includes("Safari") && !/Chrome|Chromium|Edg\//u.test(userAgent)) {
    return "desktop-safari";
  }

  return "desktop-chromium";
}

export function installSteps(platform: Platform): InstallStep[] {
  return stepsByPlatform[platform];
}
