import { useSyncExternalStore } from "react";

const tickMs = 30_000;

const listeners = new Set<() => void>();

let now = Date.now();

let timer: ReturnType<typeof setInterval> | null = null;

function tick() {
  now = Date.now();

  for (const listener of listeners) {
    listener();
  }
}

function subscribe(onChange: () => void): () => void {
  listeners.add(onChange);

  if (timer === null) {
    now = Date.now();
    timer = setInterval(tick, tickMs);
    document.addEventListener("visibilitychange", tick);
  }

  return () => {
    listeners.delete(onChange);

    if (listeners.size === 0 && timer !== null) {
      clearInterval(timer);
      timer = null;
      document.removeEventListener("visibilitychange", tick);
    }
  };
}

function getNow(): number {
  return now;
}

export function useNow(): number {
  return useSyncExternalStore(subscribe, getNow);
}
