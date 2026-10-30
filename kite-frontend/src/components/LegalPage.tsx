import type { ReactNode } from 'react';
import Link from 'next/link';
import { GlobalHeader } from './GlobalHeader';
import { AuthProvider } from '../store/useAuthStore';
import { getSession } from '../server/api';

type LegalPageProps = {
  readonly title: string;
  readonly document: 'privacy' | 'terms';
  readonly children: ReactNode;
};

// LegalPage gives public policy documents a shared reading layout and document navigation.
// document identifies the active policy; children supplies its server-rendered sections.
export async function LegalPage({ title, document, children }: LegalPageProps) {
  const user = await getSession();
  return (
    <AuthProvider key={user ? user.username + ':' + user.access_level : 'anonymous'} user={user}>
    <div className="public-page">
      <GlobalHeader />
      <main id="main-content" className="marketing-legal">
        <header className="marketing-legal-heading">
          <p className="marketing-eyebrow">Kite / Policies</p>
          <h1>{title}</h1>
          <p className="marketing-legal-date">최종 개정일 <time dateTime="2026-10-30">2026년 10월 30일</time></p>
          <nav aria-label="정책 문서">
            <Link href="/privacy" aria-current={document === 'privacy' ? 'page' : undefined}>개인정보처리방침</Link>
            <Link href="/terms" aria-current={document === 'terms' ? 'page' : undefined}>서비스 이용약관</Link>
          </nav>
        </header>
        <article className="marketing-legal-content" aria-label={title}>
          {children}
        </article>
      </main>
    </div>
    </AuthProvider>
  );
}
