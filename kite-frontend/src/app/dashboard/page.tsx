import type { Metadata } from 'next';
import { UserDashboard } from '@/screens/UserDashboard';
import { AuthProvider } from '@/store/useAuthStore';
import { getServerApi, requireSession } from '@/server/api';
import { loadDashboard } from '@/api/pageData';
import { USE_MOCK_API } from '@/config/debug';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata describes the private workspace without exposing account data.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('워크스페이스', { path: '/dashboard', noIndex: true, description: '내 가상머신의 상태와 배정된 리소스를 확인하고 관리하는 Kite 워크스페이스입니다.' });
}

// Page authorizes every SSR/RSC request before loading private initial screen data.
export default async function Page() {
  const user = await requireSession('/dashboard', 0);
  const { client } = await getServerApi();
  const data = USE_MOCK_API ? undefined : await loadDashboard(client);
  return <AuthProvider key={user ? user.username + ':' + user.access_level : 'anonymous'} user={user}><UserDashboard initialData={data} /></AuthProvider>;
}
