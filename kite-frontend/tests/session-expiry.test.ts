import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';
import { proxy } from '../src/proxy';
import { shouldClearSessionCookie } from '../src/server/sessionCookie';
import { formatSessionCountdown, remainingSessionSeconds, watchSessionExpiry } from '../src/lib/sessionCountdown';

const now = Date.parse('2030-01-01T00:00:00Z');
const future = '2030-01-01T00:10:00Z';

// token builds deliberately unsigned input: parsing here may reject but never authenticate it.
function token(exp: number): string {
  return `header.${Buffer.from(JSON.stringify({ exp })).toString('base64url')}.unsigned`;
}

beforeEach(() => { vi.useFakeTimers(); vi.setSystemTime(now); });
afterEach(() => vi.useRealTimers());

it.each([[600, '10:00'], [61, '01:01'], [0, '00:00'], [3600, '60:00']])('formats %s remaining seconds as %s', (seconds, expected) => {
  expect(formatSessionCountdown(Number(seconds))).toBe(expected);
});

it('expires once at zero and stops ticking', async () => {
  const tick = vi.fn();
  const expire = vi.fn();
  watchSessionExpiry(new Date(now + 2100).toISOString(), { tick, expire });
  await Promise.resolve();
  expect(tick).toHaveBeenLastCalledWith(3);
  vi.advanceTimersByTime(2000);
  expect(tick).toHaveBeenLastCalledWith(1);
  expect(expire).not.toHaveBeenCalled();
  vi.advanceTimersByTime(1000);
  expect(tick).toHaveBeenLastCalledWith(0);
  expect(expire).toHaveBeenCalledTimes(1);
  vi.advanceTimersByTime(60_000);
  expect(expire).toHaveBeenCalledTimes(1);
});

it('uses the absolute deadline after a suspended tab resumes', async () => {
  const expire = vi.fn();
  watchSessionExpiry(future, { tick: vi.fn(), expire });
  await Promise.resolve();
  vi.setSystemTime(now + 700_000);
  vi.advanceTimersByTime(1000);
  expect(expire).toHaveBeenCalledTimes(1);
});

it('cleanup cancels even the initial queued update', async () => {
  const tick = vi.fn();
  const expire = vi.fn();
  const stop = watchSessionExpiry(future, { tick, expire });
  stop();
  await Promise.resolve();
  vi.advanceTimersByTime(700_000);
  expect(tick).not.toHaveBeenCalled();
  expect(expire).not.toHaveBeenCalled();
});

it('rejects expired and malformed cookie claims without trusting a future signature', () => {
  expect(shouldClearSessionCookie(`"Bearer ${token(now / 1000)}"`, now)).toBe(true);
  expect(shouldClearSessionCookie('invalid', now)).toBe(true);
  expect(shouldClearSessionCookie('a.invalid.b', now)).toBe(true);
  expect(shouldClearSessionCookie(`Bearer ${token(now / 1000 + 600)}`, now)).toBe(false);
  expect(remainingSessionSeconds(future, now + 700_000)).toBe(0);
});

it('removes an expired JWT from both the SSR request and browser response', () => {
  const request = new NextRequest('http://localhost/dashboard', { headers: { cookie: `accessToken="Bearer ${token(now / 1000 - 1)}"; preference=keep` } });
  const response = proxy(request);
  expect(request.cookies.has('accessToken')).toBe(false);
  expect(request.cookies.get('preference')?.value).toBe('keep');
  expect(response.cookies.get('accessToken')?.value).toBe('');
  expect(response.headers.get('x-middleware-request-cookie')).toBe('preference=keep');
});
