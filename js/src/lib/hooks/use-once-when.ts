import { useEffect, useRef } from "react";

export function useOnceWhen(condition: boolean, run: () => void): void {
  const ran = useRef(false);

  useEffect(() => {
    if (!condition || ran.current) {
      return;
    }

    ran.current = true;
    run();
  }, [condition, run]);
}
