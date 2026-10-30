'use client';

import { createContext, useContext, useState } from 'react';
import type { ReactNode } from 'react';
import { useStore } from 'zustand';
import type { SessionUser } from '../api/session';
import { createAuthStore } from './authState';
import type { AuthState } from './authState';

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
