'use client';

import { createContext, useContext, useState } from 'react';
import type { ReactNode } from 'react';
import { useStore } from 'zustand';
import { createStore } from 'zustand/vanilla';
import type { SessionUser } from '../api/session';

type AuthState = {
  readonly authenticated: boolean;
  readonly accessLevel: number | null;
  readonly username: string | null;
  readonly email: string | null;
  readonly namespace: string | null;
  readonly login: (user: SessionUser) => void;
  readonly logout: () => void;
};

// createAuthStore isolates display state per SSR request and page provider.
// user is verified by /me; JWTs and permission claims are never read from browser storage.
function createAuthStore(user: SessionUser | null) {
  return createStore<AuthState>((set) => ({
    authenticated: user !== null, accessLevel: user?.access_level ?? null,
    username: user?.username ?? null, email: user?.email ?? null, namespace: user?.namespace ?? null,
    login: (profile) => set({ authenticated: true, accessLevel: profile.access_level, username: profile.username, email: profile.email, namespace: profile.namespace }),
    logout: () => set({ authenticated: false, accessLevel: null, username: null, email: null, namespace: null }),
  }));
}

const AuthContext = createContext<ReturnType<typeof createAuthStore> | null>(null);

// AuthProvider gives children a private store seeded with this page's verified user.
// A new provider per page prevents state leaking between SSR users or navigations.
export function AuthProvider({ user, children }: { readonly user: SessionUser | null; readonly children: ReactNode }) {
  const [store] = useState(() => createAuthStore(user));
  return <AuthContext.Provider value={store}>{children}</AuthContext.Provider>;
}

class MissingAuthProviderError extends Error { readonly name = 'MissingAuthProviderError'; }

// useAuthStore subscribes to page-local user display state.
// selector optionally narrows subscriptions; Go and the SSR DAL enforce permissions.
export function useAuthStore(): AuthState;
export function useAuthStore<T>(selector: (state: AuthState) => T): T;
export function useAuthStore<T>(selector?: (state: AuthState) => T): T | AuthState {
  const store = useContext(AuthContext);
  if (!store) throw new MissingAuthProviderError('AuthProvider is required');
  return useStore(store, (state) => selector ? selector(state) : state);
}
