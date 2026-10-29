import type { Metadata } from 'next';
import { KiteDocsPage } from '@/screens/KiteDocsPage';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata pairs the public guide with its generated VM allocation illustration.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('사용 가이드', { path: '/kite-docs', image: 'guide', description: 'Kite 계정 권한, VM 사전 할당, 가상머신 생성과 SSH Gateway 접속을 안내합니다.' });
}

// Page renders the public guide without requesting account-specific session data.
export default function Page() {
  return <KiteDocsPage />;
}
