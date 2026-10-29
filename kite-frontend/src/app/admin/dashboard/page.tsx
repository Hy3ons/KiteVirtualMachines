import type { Metadata } from 'next';
import { AdminDashboard } from '@/screens/AdminDashboard';
import { AuthProvider } from '@/store/useAuthStore';
import { getServerApi, requireSession } from '@/server/api';
import { loadAdminDashboard } from '@/api/pageData';
import { USE_MOCK_API } from '@/config/debug';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata describes administrative resource tools without indexing private pages.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('사용자 및 리소스', { path: '/admin/dashboard', noIndex: true, description: 'Kite의 사용자 권한, 가상머신과 VM 사전 할당을 관리하는 관리자 화면입니다.' });
}

// Page authorizes every SSR/RSC request before loading private initial screen data.
export default async function Page() {
  const user = await requireSession('/admin/dashboard', 2);
  const { client } = await getServerApi();
  const data = USE_MOCK_API ? undefined : await loadAdminDashboard(client, user.access_level);
  return <AuthProvider key={user ? user.username + ':' + user.access_level : 'anonymous'} user={user}><AdminDashboard initialData={data} /></AuthProvider>;
}
