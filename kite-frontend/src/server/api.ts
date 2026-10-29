import 'server-only';

import { cache } from 'react';
import { headers } from 'next/headers';
import { redirect } from 'next/navigation';
import { createApiClient } from '../api/client';
import { readSession } from '../api/session';
import { DEBUG_ADMIN_SESSION, DEBUG_DIRECT_ROUTES_ENABLED } from '../config/debug';

// getServerApi forwards only the JWT cookie from the current request to the
// internal Go API. React cache deduplicates within this server render only.
export const getServerApi = cache(async () => {
  const incoming = (await headers()).get('cookie') || '';
  const cookie = incoming.split(';').map((value) => value.trim()).find((value) => value.startsWith('accessToken=')) || '';
  return {
    client: createApiClient(`${(process.env.KITE_API_INTERNAL_URL || 'http://127.0.0.1:8080').replace(/\/$/, '')}/api/v1`, { cookie }),
    hasCookie: cookie !== '',
  };
});

// getSession returns the current backend-verified profile for SSR.
// Only explicit debug/mock builds can seed a synthetic session.
export const getSession = cache(async () => {
  if (DEBUG_DIRECT_ROUTES_ENABLED) {
    return { username: DEBUG_ADMIN_SESSION.username, email: DEBUG_ADMIN_SESSION.email, namespace: DEBUG_ADMIN_SESSION.namespace, access_level: DEBUG_ADMIN_SESSION.accessLevel, profile_image: '' };
  }
  const { client, hasCookie } = await getServerApi();
  return readSession(client, hasCookie);
});

// requireSession guards each page and data load, including RSC navigation.
// pathname is the fixed login-notice route; minimumAccessLevel is compared with
// current CRD-backed /me permissions rather than locally decoded JWT claims.
export async function requireSession(pathname: string, minimumAccessLevel = 0) {
  const user = await getSession();
  if (!user) redirect(`/login?requireLogin=${encodeURIComponent(pathname)}`);
  if (user.access_level < minimumAccessLevel) redirect('/dashboard');
  return user;
}
