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
          <p className="marketing-legal-date">운영 정책 초안 · 작성일 <time dateTime="2026-10-30">2026년 10월 30일</time></p>
          <nav aria-label="정책 문서">
            <Link href="/privacy" aria-current={document === 'privacy' ? 'page' : undefined}>개인정보처리방침</Link>
            <Link href="/terms" aria-current={document === 'terms' ? 'page' : undefined}>서비스 이용약관</Link>
          </nav>
        </header>
        <article className="marketing-legal-content" aria-label={title}>
          <aside className="marketing-legal-notice" aria-label="문서 적용 안내">
            Google 로그인 도입을 전제로 작성한 초안입니다. 운영자 정보와 문의처, 로그·백업 보유기간, 위탁·국외이전 내역을 확정한 뒤 시행일을 공지하며, 그 전에는 실제 서비스의 확정 정책으로 적용하지 않습니다.
          </aside>
          {children}
        </article>
      </main>
    </div>
    </AuthProvider>
  );
}
