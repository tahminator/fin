import { useSyncExternalStore } from "react";

const standaloneQuery = "(display-mode: standalone)";

function subscribe(onChange: () => void): () => void {
  const media = globalThis.matchMedia(standaloneQuery);

  media.addEventListener("change", onChange);

  return () => {
    media.removeEventListener("change", onChange);
  };
}

function isStandalone(): boolean {
  return globalThis.matchMedia(standaloneQuery).matches || navigator.standalone === true;
}

export function useIsStandalone(): boolean {
  return useSyncExternalStore(subscribe, isStandalone);
}
