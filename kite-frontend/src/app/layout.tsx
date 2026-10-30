import type { Metadata } from 'next';
import type { ReactNode } from 'react';
import { AntdRegistry } from '@ant-design/nextjs-registry';
import { Providers } from './providers';
import { GlobalFooter } from '../components/GlobalFooter';
import { createPageMetadata, HOME_TITLE } from '../lib/metadata';
import { getPublicOrigin } from '../server/publicOrigin';
import '../index.css';

// generateMetadata resolves public URLs at request time when no deployment origin is set.
// Explicit KITE_PUBLIC_URL also supports prerendered public metadata at build time.
export async function generateMetadata(): Promise<Metadata> {
  return {
    ...await createPageMetadata('Kite', { home: true, path: '/' }),
    metadataBase: await getPublicOrigin(),
    title: { default: HOME_TITLE, template: '%s | Kite' },
    icons: {
      icon: [{ url: '/favicon.svg?v=3', type: 'image/svg+xml' }, { url: '/favicon.ico?v=3', sizes: '32x32' }],
      apple: [{ url: '/apple-touch-icon.png?v=3', sizes: '180x180', type: 'image/png' }],
    },
    authors: [{ name: 'Hy3ons' }],
    keywords: ['Kite', 'Kubernetes', 'KubeVirt', 'Virtual Machine', '클라우드', '가상머신'],
  };
}

// RootLayout renders the shared document and extracts Ant Design SSR styles.
// children is the route selected by Next's App Router.
export default function RootLayout({ children }: { readonly children: ReactNode }) {
  return (
    <html lang="ko">
      <head><link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/orioncactus/pretendard@v1.3.9/dist/web/variable/pretendardvariable.min.css" /></head>
      <body><a className="skip-link" href="#main-content">본문으로 건너뛰기</a><AntdRegistry><Providers>{children}<GlobalFooter /></Providers></AntdRegistry></body>
    </html>
  );
}
