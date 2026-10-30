import type { Metadata } from 'next';
import { VmConsolePage } from '@/screens/VmConsolePage';
import { AuthProvider } from '@/store/useAuthStore';
import { getServerApi, requireSession } from '@/server/api';
import { loadConsoleVm } from '@/api/pageData';
import { USE_MOCK_API } from '@/config/debug';
import { createPageMetadata } from '@/lib/metadata';

type Props = { readonly params: Promise<{ readonly vmName: string }> };

// generateMetadata names the requested VM without exposing private VM data.
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { vmName } = await params;
  return createPageMetadata(`${vmName} | 웹 콘솔`, { path: '/dashboard/kite-machine/' + encodeURIComponent(vmName) + '/console', noIndex: true, description: '브라우저에서 가상머신 콘솔에 접속하는 Kite 웹 콘솔입니다.' });
}

// Page verifies the session before SSR-loading VM metadata; params selects the VM.
export default async function Page({ params }: Props) {
  const { vmName } = await params;
  const user = await requireSession('/dashboard/kite-machine/' + encodeURIComponent(vmName) + '/console');
  const { client } = await getServerApi();
  const data = USE_MOCK_API ? null : await loadConsoleVm(client, vmName);
  return <AuthProvider key={vmName + ':' + user.username + ':' + user.access_level} user={user}><VmConsolePage initialVm={data?.vm ?? null} /></AuthProvider>;
}
