'use client';

import { useEffect, useState } from 'react';
import { useAuthStore } from '../store/useAuthStore';
import { useLogout } from '../hooks/useLogout';
import { formatSessionCountdown, watchSessionExpiry } from '../lib/sessionCountdown';

// SessionCountdown shares the session deadline across public and console headers.
// A stable initial placeholder avoids SSR hydration drift; no JWT is exposed.
export function SessionCountdown() {
  const expiresAt = useAuthStore((state) => state.sessionExpiresAt);
  const logout = useLogout('expired');
  const [remaining, setRemaining] = useState<number | null>(null);
  useEffect(() => {
    if (!expiresAt) return;
    return watchSessionExpiry(expiresAt, { tick: setRemaining, expire: () => { void logout(); } });
  }, [expiresAt, logout]);
  if (!expiresAt) return null;
  return <span className={`session-countdown${remaining !== null && remaining <= 60 ? ' is-expiring' : ''}`} role="timer" aria-live="off" aria-label="로그인 세션 만료까지 남은 시간">
    <span className="session-countdown-label">만료까지</span>
    <span>{remaining === null ? '--:--' : formatSessionCountdown(remaining)}</span>
  </span>;
}
