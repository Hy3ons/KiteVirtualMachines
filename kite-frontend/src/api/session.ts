import type { z } from 'zod';
import { ApiError } from './client';
import type { ApiClient } from './client';

import { createEndpoints } from './endpoints';
import type { sessionUserSchema } from './contracts';
export { sessionUserSchema, authResponseSchema } from './contracts';
export type SessionUser = z.infer<typeof sessionUserSchema> & { readonly sessionExpiresAt: string | null };

// readSession asks client to validate an existing JWT cookie through Go's /me.
// hasCookie skips anonymous lookups. Invalid/deleted sessions return null;
// outages and malformed responses propagate instead of pretending the user logged out.
export async function readSession(client: ApiClient, hasCookie: boolean): Promise<SessionUser | null> {
  if (!hasCookie) return null;
  try {
    const { user, expiresAt } = await createEndpoints(client).authApi.getMe();
    if (Date.parse(expiresAt) <= Date.now()) return null;
    return { ...user, sessionExpiresAt: expiresAt };
  } catch (error) {
    if (error instanceof ApiError && [401, 403, 404].includes(error.status)) return null;
    throw error;
  }
}
