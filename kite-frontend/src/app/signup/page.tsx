import type { Metadata } from 'next';
import { SignupPage } from '@/screens/SignupPage';
import { AuthProvider } from '@/store/useAuthStore';
import { getSession } from '@/server/api';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata describes registration without indexing an account utility page.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('회원가입', { path: '/signup', noIndex: true, description: 'Kite 계정을 만들고 관리자가 허용한 가상머신과 리소스를 사용하세요.' });
}

// Page server-renders the existing screen with the current backend-verified session.
export default async function Page() {
  const user = await getSession();
  return <AuthProvider key={user ? user.username + ':' + user.access_level : 'anonymous'} user={user}><SignupPage /></AuthProvider>;
}
