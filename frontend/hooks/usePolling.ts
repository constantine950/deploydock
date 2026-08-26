import { useEffect, useRef } from "react";

export function usePolling(
  callback: () => void,
  interval: number,
  active = true,
) {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  useEffect(() => {
    if (!active) return;

    const tick = () => callbackRef.current();
    tick();

    const id = setInterval(tick, interval);
    return () => clearInterval(id);
  }, [interval, active]);
}
