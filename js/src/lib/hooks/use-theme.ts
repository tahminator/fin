import { useEffect, useState } from "react";

type ThemePreference = "dark" | "light" | "system";

const storageKey = "fin-theme";

function getStoredPreference(): ThemePreference {
  const stored = globalThis.localStorage.getItem(storageKey);

  return stored === "dark" || stored === "light" || stored === "system" ? stored : "system";
}

export function useThemeState() {
  const [state, setState] = useState(() => ({
    preference: getStoredPreference(),
    systemDark: globalThis.matchMedia("(prefers-color-scheme: dark)").matches,
  }));

  useEffect(() => {
    const mediaQuery = globalThis.matchMedia("(prefers-color-scheme: dark)");
    const onChange = (event: MediaQueryListEvent) => {
      setState((current) => ({ ...current, systemDark: event.matches }));
    };

    mediaQuery.addEventListener("change", onChange);

    return () => {
      mediaQuery.removeEventListener("change", onChange);
    };
  }, []);

  useEffect(() => {
    const isDark =
      state.preference === "dark" || (state.preference === "system" && state.systemDark);

    globalThis.document.documentElement.classList.toggle("dark", isDark);
    globalThis.document.documentElement.style.colorScheme = isDark ? "dark" : "light";
  }, [state]);

  return {
    preference: state.preference,
    setPreference: (preference: ThemePreference) => {
      globalThis.localStorage.setItem(storageKey, preference);
      setState((current) => ({ ...current, preference }));
    },
  };
}
