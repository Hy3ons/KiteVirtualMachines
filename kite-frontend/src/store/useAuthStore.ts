import { create } from 'zustand';
import { seedDebugAuthSession } from '../config/debug';

interface AuthState {
  authenticated: boolean;
  accessLevel: number | null;
  username: string | null;
  namespace: string | null;
  login: (accessLevel: number, username: string, namespace: string) => void;
  logout: () => void;
}

seedDebugAuthSession(localStorage);

const initialAuthenticated = localStorage.getItem('kite_authenticated') === 'true';
const initialAccessLevel = localStorage.getItem('kite_access_level');
const initialUsername = localStorage.getItem('kite_username');
const initialNamespace = localStorage.getItem('kite_namespace');

export const useAuthStore = create<AuthState>((set) => ({
  authenticated: initialAuthenticated,
  accessLevel: initialAccessLevel ? parseInt(initialAccessLevel, 10) : null,
  username: initialUsername,
  namespace: initialNamespace,
  
  login: (accessLevel, username, namespace) => {
    localStorage.setItem('kite_authenticated', 'true');
    localStorage.setItem('kite_access_level', accessLevel.toString());
    localStorage.setItem('kite_username', username);
    localStorage.setItem('kite_namespace', namespace);
    localStorage.removeItem('kite_profile_image');
    set({ authenticated: true, accessLevel, username, namespace });
  },
  
  logout: () => {
    localStorage.removeItem('kite_authenticated');
    localStorage.removeItem('kite_access_level');
    localStorage.removeItem('kite_username');
    localStorage.removeItem('kite_namespace');
    localStorage.removeItem('kite_profile_image');
    set({ authenticated: false, accessLevel: null, username: null, namespace: null });
  },
}));
