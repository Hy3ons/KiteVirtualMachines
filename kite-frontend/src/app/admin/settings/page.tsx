import type { Metadata } from 'next';
import { AdminSettings } from '@/screens/AdminSettings';
import { AuthProvider } from '@/store/useAuthStore';
import { getServerApi, requireSession } from '@/server/api';
import { loadSettings } from '@/api/pageData';
import { USE_MOCK_API } from '@/config/debug';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata describes operational settings without indexing private pages.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('시스템 설정', { path: '/admin/settings', noIndex: true, description: 'Kite 도메인, HTTPS, SSH Gateway와 런타임 설정을 관리하는 관리자 화면입니다.' });
}

// Page authorizes every SSR/RSC request before loading private initial screen data.
export default async function Page() {
  const user = await requireSession('/admin/settings', 3);
  const { client } = await getServerApi();
  const data = USE_MOCK_API ? undefined : await loadSettings(client);
  return <AuthProvider key={user ? user.username + ':' + user.access_level : 'anonymous'} user={user}><AdminSettings initialData={data} /></AuthProvider>;
}
