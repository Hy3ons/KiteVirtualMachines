import type { Metadata } from 'next';
import { VmDetail } from '@/screens/VmDetail';
import { AuthProvider } from '@/store/useAuthStore';
import { getServerApi, requireSession } from '@/server/api';
import { loadVm } from '@/api/pageData';
import { USE_MOCK_API } from '@/config/debug';
import { createPageMetadata } from '@/lib/metadata';

type Props = { readonly params: Promise<{ readonly vmName: string }> };

// generateMetadata names the requested VM without exposing private VM data.
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { vmName } = await params;
  return createPageMetadata(`${vmName} | 가상머신`, { path: '/dashboard/kite-machine/' + encodeURIComponent(vmName), noIndex: true, description: '가상머신의 구성, 상태와 접속 정보를 확인하는 Kite 상세 화면입니다.' });
}

// Page verifies the session before SSR-loading VM metadata; params selects the VM.
export default async function Page({ params }: Props) {
  const { vmName } = await params;
  const user = await requireSession('/dashboard/kite-machine/' + encodeURIComponent(vmName));
  const { client } = await getServerApi();
  const data = USE_MOCK_API ? undefined : await loadVm(client, vmName);
  return <AuthProvider key={vmName + ':' + user.username + ':' + user.access_level} user={user}><VmDetail initialData={data} /></AuthProvider>;
}
