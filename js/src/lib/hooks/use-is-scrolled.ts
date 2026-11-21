import { useSyncExternalStore } from "react";

function subscribe(onChange: () => void): () => void {
  globalThis.addEventListener("scroll", onChange, { passive: true });

  return () => {
    globalThis.removeEventListener("scroll", onChange);
  };
}

function isScrolled(): boolean {
  return globalThis.scrollY > 0;
}

export function useIsScrolled(): boolean {
  return useSyncExternalStore(subscribe, isScrolled);
}
