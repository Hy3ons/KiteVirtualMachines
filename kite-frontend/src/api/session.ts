import { z } from 'zod';
import { ApiError } from './client';
import type { ApiClient } from './client';

export const sessionUserSchema = z.object({
  username: z.string().min(1), email: z.string(), namespace: z.string().min(1),
  access_level: z.number().int().min(0).max(3), profile_image: z.string().default(''),
});
export type SessionUser = z.infer<typeof sessionUserSchema>;
const meSchema = z.object({ user: sessionUserSchema });
export const authResponseSchema = z.object({
  user: sessionUserSchema,
  expiresIn: z.number(),
  expiresAt: z.string(),
});

// readSession asks client to validate an existing JWT cookie through Go's /me.
// hasCookie skips anonymous lookups. Invalid/deleted sessions return null;
// outages and malformed responses propagate instead of pretending the user logged out.
export async function readSession(client: ApiClient, hasCookie: boolean): Promise<SessionUser | null> {
  if (!hasCookie) return null;
  try {
    const { data } = await client.get<unknown>('/me');
    return meSchema.parse(data).user;
  } catch (error) {
    if (error instanceof ApiError && [401, 403, 404].includes(error.status)) return null;
    throw error;
  }
}
