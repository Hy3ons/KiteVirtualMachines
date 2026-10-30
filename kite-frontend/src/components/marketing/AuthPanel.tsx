'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { App, Button } from 'antd';
import { ArrowRightOutlined } from '@ant-design/icons';
import { authApi } from '../../api';
import { useAuthStore } from '../../store/useAuthStore';
import { canManageUsers } from '../../domain/access';
import { useLogout } from '../../hooks/useLogout';
import { LoginForm } from './LoginForm';
import type { LoginFormValues } from './LoginForm';

// AuthPanel renders the dedicated login page's Go-backed form or current session actions.
// Cookie ownership stays with the API; successful login refreshes the protected Next route.
export function AuthPanel() {
  const { message, notification } = App.useApp();
  const { login, authenticated, username, accessLevel } = useAuthStore();
  const logout = useLogout();
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

  // submitLogin sends the labelled email and password to Go, then updates display state.
  // values come from Ant Design's validated form; failures keep the user on the form.
  const submitLogin = async (values: LoginFormValues) => {
    try {
      setLoading(true);
      const { user } = await authApi.login({ email: values.username, password: values.password });
      login(user);
      message.success('로그인 되었습니다.');
      router.push('/dashboard');
      router.refresh();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '로그인에 실패했습니다.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="marketing-auth-panel">
      {authenticated ? (
        <div className="marketing-session">
          <span className="marketing-eyebrow">Your workspace is ready</span>
          <h1>{username}님,<br />다시 만나 반가워요.</h1>
          <p>로그인된 계정으로 작업을 이어가세요.</p>
          <Link href="/dashboard" className="marketing-button marketing-button--primary">내 가상머신으로 <ArrowRightOutlined /></Link>
          {canManageUsers(accessLevel) && (
            <Link href="/admin/dashboard" className="marketing-button marketing-button--secondary">Admin Console</Link>
          )}
          <Button type="text" onClick={logout} block>로그아웃</Button>
        </div>
      ) : (
        <>
          <div className="marketing-form-heading"><h1>Kite에 로그인</h1></div>
          <LoginForm onFinish={submitLogin} loading={loading} />
          <p className="marketing-form-footer">아직 계정이 없으신가요? <Link href="/signup">회원가입</Link></p>
        </>
      )}
    </div>
  );
}
