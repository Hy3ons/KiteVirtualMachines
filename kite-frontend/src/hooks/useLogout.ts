import { useCallback } from 'react';
import { authApi } from '../api';
import { useAuthStore } from '../store/useAuthStore';

export const useLogout = () => {
  const clearAuth = useAuthStore((state) => state.logout);

  return useCallback(() => {
    void authApi.logout().then(clearAuth, clearAuth);
  }, [clearAuth]);
};
