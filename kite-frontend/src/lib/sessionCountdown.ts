// remainingSessionSeconds derives whole seconds from a validated API expiry
// and the current clock, without extending a session after tab suspension.
export function remainingSessionSeconds(expiresAt: string, now: number): number {
  return Math.max(0, Math.ceil((Date.parse(expiresAt) - now) / 1000));
}

// formatSessionCountdown presents total minutes and seconds, including durations
// over an hour. seconds is the non-negative result of remainingSessionSeconds.
export function formatSessionCountdown(seconds: number): string {
  return `${Math.floor(seconds / 60).toString().padStart(2, '0')}:${(seconds % 60).toString().padStart(2, '0')}`;
}

type CountdownActions = {
  readonly tick: (seconds: number) => void;
  readonly expire: () => void;
};

// watchSessionExpiry updates actions from the actual clock, expires exactly once,
// and rechecks on focus/visibility changes. expiresAt comes from the verified API.
// The returned cleanup cancels timers, listeners and a pending initial tick.
export function watchSessionExpiry(expiresAt: string, actions: CountdownActions): () => void {
  let active = true;
  const stop = () => {
    active = false;
    clearInterval(timer);
    if (typeof window !== 'undefined') window.removeEventListener('focus', update);
    if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', update);
  };
  const update = () => {
    if (!active) return;
    const seconds = remainingSessionSeconds(expiresAt, Date.now());
    actions.tick(seconds);
    if (seconds === 0) {
      stop();
      actions.expire();
    }
  };
  const timer = setInterval(update, 1000);
  if (typeof window !== 'undefined') window.addEventListener('focus', update);
  if (typeof document !== 'undefined') document.addEventListener('visibilitychange', update);
  queueMicrotask(update);
  return stop;
}
