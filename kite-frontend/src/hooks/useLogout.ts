'use client';

import { useCallback } from 'react';
import { App } from 'antd';
import { authApi } from '../api';
import { useAuthStore } from '../store/useAuthStore';

// useLogout removes the API-owned cookie and requests a fresh anonymous page.
// reason distinguishes an explicit logout from an already expired session.
// Expiry discards display state even if Go is unavailable; SSR also removes expired cookies.
export const useLogout = (reason: 'manual' | 'expired' = 'manual') => {
  const clearAuth = useAuthStore((state) => state.logout);
  const { message } = App.useApp();
  return useCallback(async () => {
    if (reason === 'expired') clearAuth();
    try {
      await authApi.logout();
    } catch (error) {
      if (!(error instanceof Error)) throw error;
      if (reason === 'manual') { message.error('로그아웃에 실패했습니다. 다시 시도해 주세요.'); return; }
    }
    clearAuth();
    window.location.replace(reason === 'expired' ? '/login?sessionExpired=true' : '/');
  }, [clearAuth, message, reason]);
};
