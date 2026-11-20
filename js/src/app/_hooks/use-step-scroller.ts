import { useRef, useState } from "react";

export function useStepScroller(stepCount: number) {
  const trackRef = useRef<HTMLDivElement>(null);
  const [step, setStep] = useState(0);

  const onScroll = () => {
    const track = trackRef.current;

    if (track === null || track.clientWidth === 0) {
      return;
    }

    setStep(Math.round(track.scrollLeft / track.clientWidth));
  };

  const goTo = (index: number) => {
    const track = trackRef.current;
    const target = Math.min(Math.max(index, 0), stepCount - 1);

    track?.scrollTo({ left: target * track.clientWidth, behavior: "smooth" });
  };

  return {
    trackRef,
    step,
    onScroll,
    goTo,
    isFirst: step === 0,
    isLast: step === stepCount - 1,
  };
}
