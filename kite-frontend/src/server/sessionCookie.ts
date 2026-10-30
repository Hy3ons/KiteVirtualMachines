import { z } from 'zod';

const expiryClaims = z.object({ exp: z.number().int().nonnegative() });

// shouldClearSessionCookie rejects malformed or expired cookies before SSR.
// value is untrusted cookie text and now is the server clock in milliseconds.
// This only rejects sessions: Go still verifies the signature and current user.
export function shouldClearSessionCookie(value: string, now: number): boolean {
  const trimmed = value.trim();
  const token = (trimmed.startsWith('"') && trimmed.endsWith('"') ? trimmed.slice(1, -1) : trimmed).replace(/^Bearer\s+/, '');
  const parts = token.split('.');
  if (parts.length !== 3 || !parts[1]) return true;
  try {
    const claims = expiryClaims.safeParse(JSON.parse(Buffer.from(parts[1], 'base64url').toString('utf8')));
    return !claims.success || claims.data.exp * 1000 <= now;
  } catch (error) {
    if (error instanceof SyntaxError) return true;
    throw error;
  }
}
