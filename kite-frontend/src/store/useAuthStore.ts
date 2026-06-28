import { create } from 'zustand';
import { seedDebugAuthSession } from '../config/debug';

interface AuthState {
  authenticated: boolean;
  accessLevel: number | null;
  username: string | null;
  namespace: string | null;
  profileImage: string | null;
  login: (accessLevel: number, username: string, namespace: string, profileImage: string) => void;
  logout: () => void;
}

seedDebugAuthSession(localStorage);

const initialAuthenticated = localStorage.getItem('kite_authenticated') === 'true';
const initialAccessLevel = localStorage.getItem('kite_access_level');
const initialUsername = localStorage.getItem('kite_username');
const initialNamespace = localStorage.getItem('kite_namespace');
const initialProfileImage = localStorage.getItem('kite_profile_image');

export const useAuthStore = create<AuthState>((set) => ({
  authenticated: initialAuthenticated,
  accessLevel: initialAccessLevel ? parseInt(initialAccessLevel, 10) : null,
  username: initialUsername,
  namespace: initialNamespace,
  profileImage: initialProfileImage,
  
  login: (accessLevel, username, namespace, profileImage) => {
    localStorage.setItem('kite_authenticated', 'true');
    localStorage.setItem('kite_access_level', accessLevel.toString());
    localStorage.setItem('kite_username', username);
    localStorage.setItem('kite_namespace', namespace);
    localStorage.setItem('kite_profile_image', profileImage || '');
    set({ authenticated: true, accessLevel, username, namespace, profileImage });
  },
  
  logout: () => {
    localStorage.removeItem('kite_authenticated');
    localStorage.removeItem('kite_access_level');
    localStorage.removeItem('kite_username');
    localStorage.removeItem('kite_namespace');
    localStorage.removeItem('kite_profile_image');
    set({ authenticated: false, accessLevel: null, username: null, namespace: null, profileImage: null });
  },
}));
