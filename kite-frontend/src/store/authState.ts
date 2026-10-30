import { createStore } from 'zustand/vanilla';
import type { SessionUser } from '../api/session';

type AnonymousProfile = {
  readonly authenticated: false;
  readonly accessLevel: null;
  readonly username: null;
  readonly email: null;
  readonly namespace: null;
};

type AuthenticatedProfile = {
  readonly authenticated: true;
  readonly accessLevel: SessionUser['access_level'];
  readonly username: string;
  readonly email: string;
  readonly namespace: string;
};

type AuthProfile = AnonymousProfile | AuthenticatedProfile;
export type AuthState = AuthProfile & {
  readonly login: (user: SessionUser) => void;
  readonly logout: () => void;
};

// profileFor derives the complete display snapshot from a backend-verified user.
// null represents an anonymous session, so identity and permissions change together.
function profileFor(user: SessionUser | null): AuthProfile {
  if (user === null) return { authenticated: false, accessLevel: null, username: null, email: null, namespace: null };
  return { authenticated: true, accessLevel: user.access_level, username: user.username, email: user.email, namespace: user.namespace };
}

// createAuthStore isolates display state per SSR request and page provider.
// user comes from /me; neither JWTs nor permission claims use browser storage.
export function createAuthStore(user: SessionUser | null) {
  return createStore<AuthState>((set) => ({
    ...profileFor(user),
    login: (profile) => set(profileFor(profile)),
    logout: () => set(profileFor(null)),
  }));
}
