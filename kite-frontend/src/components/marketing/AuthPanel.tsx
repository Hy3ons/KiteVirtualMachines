'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { App } from 'antd';
import { authApi } from '../../api';
import { useAuthStore } from '../../store/useAuthStore';
import { LoginForm } from './LoginForm';
import type { LoginFormValues } from './LoginForm';

// AuthPanel renders the dedicated login page's Go-backed form.
// Successful login requests the dashboard with the new API-owned session cookie.
export function AuthPanel() {
  const { message, notification } = App.useApp();
  const login = useAuthStore((state) => state.login);
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const query = new URLSearchParams(window.location.search);
    const requiredPath = query.get('requireLogin');
    if (requiredPath || query.has('sessionExpired')) {
      notification.warning({
        title: '로그인이 필요한 서비스입니다.',
        description: requiredPath ? '접근하시려는 페이지는 로그인 후 이용하실 수 있습니다.' : '세션이 만료되었습니다. 다시 로그인해 주세요.',
        placement: 'top', duration: 4,
      });
      router.replace('/login', { scroll: false });
    }
  }, [notification, router]);

  // submitLogin sends validated form values to Go and navigates directly to the dashboard.
  // A full navigation avoids refreshing the login route with the newly issued cookie.
  // Failed requests keep the form available with an error message.
  const submitLogin = async (values: LoginFormValues) => {
    try {
      setLoading(true);
      const { user } = await authApi.login({ email: values.username, password: values.password });
      login(user);
      window.location.replace('/dashboard');
    } catch (error) {
      setLoading(false);
      message.error(error instanceof Error ? error.message : '로그인에 실패했습니다.');
    }
  };

  return (
    <div className="marketing-auth-panel">
      <div className="marketing-form-heading"><h1>Kite에 로그인</h1></div>
      <LoginForm onFinish={submitLogin} loading={loading} />
      <p className="marketing-form-footer">아직 계정이 없으신가요? <Link href="/signup">회원가입</Link></p>
    </div>
  );
}
