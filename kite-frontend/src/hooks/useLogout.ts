'use client';

import { useCallback } from 'react';
import { App } from 'antd';
import { useRouter } from 'next/navigation';
import { authApi } from '../api';
import { useAuthStore } from '../store/useAuthStore';

// useLogout clears display state only after Go has removed the HttpOnly JWT cookie.
// It refreshes server-rendered data; failures stay visible so the user can retry.
export const useLogout = () => {
  const clearAuth = useAuthStore((state) => state.logout);
  const router = useRouter();
  const { message } = App.useApp();
  return useCallback(async () => {
    try {
      await authApi.logout();
      clearAuth();
      router.replace('/');
      router.refresh();
    } catch (error) {
      if (error instanceof Error) { message.error('로그아웃에 실패했습니다. 다시 시도해 주세요.'); return; }
      throw error;
    }
  }, [clearAuth, message, router]);
};
