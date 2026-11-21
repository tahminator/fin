import { MonitorIcon, MoonIcon, SunIcon } from "lucide-react";
import { createContext, useContext } from "react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useThemeState } from "@/lib/hooks/use-theme";

interface ThemeContextValue {
  preference: "dark" | "light" | "system";
  setPreference: (preference: "dark" | "light" | "system") => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const theme = useThemeState();

  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

function useTheme() {
  const theme = useContext(ThemeContext);

  if (theme === null) {
    throw new Error("useTheme must be used within ThemeProvider");
  }

  return theme;
}

const themeOptions = [
  { value: "light", icon: SunIcon, label: "Light theme" },
  { value: "dark", icon: MoonIcon, label: "Dark theme" },
  { value: "system", icon: MonitorIcon, label: "System theme" },
] as const;

export function ThemeToggle() {
  const { preference, setPreference } = useTheme();
  const CurrentIcon =
    themeOptions.find((option) => option.value === preference)?.icon ?? MonitorIcon;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="icon-lg"
          className="text-foreground"
          aria-label="Choose color theme"
        >
          <CurrentIcon />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-auto flex-row gap-1 rounded-full p-1.5">
        {themeOptions.map(({ value, icon: Icon, label }) => (
          <Button
            key={value}
            variant="ghost"
            size="icon"
            className="text-foreground aria-pressed:bg-fill aria-pressed:text-primary"
            aria-label={label}
            aria-pressed={preference === value}
            onClick={() => {
              setPreference(value);
            }}
          >
            <Icon />
          </Button>
        ))}
      </PopoverContent>
    </Popover>
  );
}
