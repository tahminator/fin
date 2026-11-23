import { useRef } from "react";
import type { TouchEvent, TransitionEvent } from "react";

const edgeWidth = 32;
const lockDistance = 8;
const backThreshold = 0.33;
const settle = "transform 280ms cubic-bezier(0.2, 0.8, 0.2, 1)";

interface Drag {
  startX: number;
  startY: number;
  offset: number;
  active: boolean;
}

function prefersReducedMotion(): boolean {
  return globalThis.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

export function useSwipeBack(onBack: () => void) {
  const panelRef = useRef<HTMLDivElement>(null);
  const drag = useRef<Drag | null>(null);
  const leaving = useRef(false);

  function setOffset(offset: number, animate: boolean) {
    const panel = panelRef.current;

    if (panel === null) {
      return;
    }

    panel.style.transition = animate ? settle : "none";
    panel.style.transform = offset === 0 ? "" : `translateX(${offset}px)`;
  }

  function back() {
    if (leaving.current) {
      return;
    }

    leaving.current = true;

    if (prefersReducedMotion() || panelRef.current === null) {
      onBack();

      return;
    }

    setOffset(panelRef.current.offsetWidth, true);
  }

  function onTouchStart(event: TouchEvent<HTMLDivElement>) {
    if (event.touches.length !== 1) {
      drag.current = null;

      return;
    }

    const touch = event.touches.item(0);

    drag.current =
      touch.clientX > edgeWidth || leaving.current
        ? null
        : { startX: touch.clientX, startY: touch.clientY, offset: 0, active: false };
  }

  function onTouchMove(event: TouchEvent<HTMLDivElement>) {
    const { current } = drag;

    if (current === null || event.touches.length === 0) {
      return;
    }

    const touch = event.touches.item(0);

    const dx = touch.clientX - current.startX;
    const dy = touch.clientY - current.startY;

    if (!current.active) {
      if (Math.abs(dx) < lockDistance && Math.abs(dy) < lockDistance) {
        return;
      }

      if (Math.abs(dy) > Math.abs(dx) || dx < 0) {
        drag.current = null;

        return;
      }

      current.active = true;
    }

    current.offset = Math.max(0, dx);
    setOffset(current.offset, false);
  }

  function onTouchEnd() {
    const { current } = drag;
    drag.current = null;

    if (current === null || !current.active || panelRef.current === null) {
      return;
    }

    if (current.offset > panelRef.current.offsetWidth * backThreshold) {
      back();
    } else {
      setOffset(0, true);
    }
  }

  function onTransitionEnd(event: TransitionEvent<HTMLDivElement>) {
    if (event.target === panelRef.current && leaving.current) {
      onBack();
    }
  }

  return {
    panelRef,
    back,
    handlers: { onTouchStart, onTouchMove, onTouchEnd, onTouchCancel: onTouchEnd, onTransitionEnd },
  };
}
