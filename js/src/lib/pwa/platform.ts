export type Platform =
  | "ios-safari"
  | "ios-other"
  | "android-chrome"
  | "desktop-chromium"
  | "desktop-safari";

export interface InstallStep {
  title: string;
  description: string;
}

const whyStep: InstallStep = {
  title: "fin runs as an installed app",
  description:
    "Add fin to your device to sign in. It opens full screen and keeps your session separate from the browser.",
};

const iosSafariSteps: InstallStep[] = [
  whyStep,
  {
    title: "Tap Share",
    description: "It's the square with an arrow pointing up, in Safari's toolbar.",
  },
  {
    title: "Tap Add to Home Screen",
    description: "Scroll down the share sheet until you see it.",
  },
  {
    title: "Tap Add, then open fin",
    description: "Launch fin from your Home Screen and sign in there.",
  },
];

const stepsByPlatform: Record<Platform, InstallStep[]> = {
  "ios-safari": iosSafariSteps,
  "ios-other": [
    whyStep,
    {
      title: "Open this page in Safari",
      description: "Only Safari can add apps to the Home Screen on iPhone and iPad.",
    },
    ...iosSafariSteps.slice(1),
  ],
  "android-chrome": [
    whyStep,
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
    whyStep,
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
    whyStep,
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
