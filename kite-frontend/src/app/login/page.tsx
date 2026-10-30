import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { LoginPage } from '@/screens/LoginPage';
import { AuthProvider } from '@/store/useAuthStore';
import { getSession } from '@/server/api';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata describes account access while excluding this utility page from search.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('로그인', { path: '/login', noIndex: true, description: 'Kite 계정으로 로그인하고 내 가상머신과 작업 공간에 접속하세요.' });
}

// Login redirects an existing Go-verified session home before rendering the anonymous login form.
export default async function Login() {
  const user = await getSession();
  if (user) redirect('/');
  return <AuthProvider user={null}><LoginPage /></AuthProvider>;
}
