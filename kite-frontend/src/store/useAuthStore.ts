import { create } from 'zustand';

interface AuthState {
  token: string | null;
  accessLevel: number | null;
  username: string | null;
  namespace: string | null;
  profileImage: string | null;
  login: (token: string, accessLevel: number, username: string, namespace: string, profileImage: string) => void;
  logout: () => void;
}

// 렌더링 시점에 localStorage를 동기적으로 읽어와 초기값을 세팅합니다.
// 이렇게 하면 JS가 실행되는 즉시 로그인 여부를 판단하여 Skeleton 또는 본 화면을 깜빡임 없이 보여줄 수 있습니다.
const initialToken = localStorage.getItem('kite_token');
const initialAccessLevel = localStorage.getItem('kite_access_level');
const initialUsername = localStorage.getItem('kite_username');
const initialNamespace = localStorage.getItem('kite_namespace');
const initialProfileImage = localStorage.getItem('kite_profile_image');

export const useAuthStore = create<AuthState>((set) => ({
  token: initialToken,
  accessLevel: initialAccessLevel ? parseInt(initialAccessLevel, 10) : null,
  username: initialUsername,
  namespace: initialNamespace,
  profileImage: initialProfileImage,
  
  login: (token, accessLevel, username, namespace, profileImage) => {
    localStorage.setItem('kite_token', token);
    localStorage.setItem('kite_access_level', accessLevel.toString());
    localStorage.setItem('kite_username', username);
    localStorage.setItem('kite_namespace', namespace);
    localStorage.setItem('kite_profile_image', profileImage || '');
    set({ token, accessLevel, username, namespace, profileImage });
  },
  
  logout: () => {
    localStorage.removeItem('kite_token');
    localStorage.removeItem('kite_access_level');
    localStorage.removeItem('kite_username');
    localStorage.removeItem('kite_namespace');
    localStorage.removeItem('kite_profile_image');
    set({ token: null, accessLevel: null, username: null, namespace: null, profileImage: null });
  },
}));
