import { createApiClient } from './client';

export const apiClient = createApiClient(process.env.NEXT_PUBLIC_API_BASE_URL || '/api/v1', {
  onUnauthorized: () => {
    // A full navigation discards stale client state and lets SSR verify the cookie again.
    if (typeof window !== 'undefined') window.location.assign('/login?sessionExpired=true');
  },
});
