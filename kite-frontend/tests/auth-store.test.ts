import { describe, expect, it } from 'vitest';
import type { SessionUser } from '../src/api/session';
import { createAuthStore } from '../src/store/authState';

const profile: SessionUser = { username: 'member', email: 'member@example.test', namespace: 'member-ns', access_level: 1, profile_image: '', sessionExpiresAt: '2030-01-01T00:00:00Z' };

describe('page-local authentication state', () => {
  it('changes the full identity when an anonymous session logs in', () => {
    // Given an anonymous page store.
    const store = createAuthStore(null);
    // When a verified profile logs in.
    store.getState().login(profile);
    // Then identity and permissions belong to that profile in one snapshot.
    expect(store.getState()).toMatchObject({ authenticated: true, username: profile.username, email: profile.email, namespace: profile.namespace, accessLevel: 1 });
  });

  it('clears identity and permissions together when the session logs out', () => {
    // Given a verified member.
    const store = createAuthStore(profile);
    // When the session ends.
    store.getState().logout();
    // Then no account fields survive in the anonymous state.
    expect(store.getState()).toMatchObject({ authenticated: false, username: null, email: null, namespace: null, accessLevel: null, sessionExpiresAt: null });
  });

  it('keeps a different SSR request isolated when one store changes', () => {
    // Given independent page stores with different identities.
    const memberStore = createAuthStore(profile);
    const adminStore = createAuthStore({ ...profile, username: 'operator', access_level: 3 });
    // When only the member signs out.
    memberStore.getState().logout();
    // Then the other request retains its own identity and permissions.
    expect(adminStore.getState()).toMatchObject({ authenticated: true, username: 'operator', accessLevel: 3 });
  });
});
