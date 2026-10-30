'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { GithubOutlined } from '@ant-design/icons';
import { Brand } from './Brand';

// GlobalFooter appears on public pages; the operational workspace has its own rail.
// Path selection avoids mounting account-dependent components outside AuthProvider.
export function GlobalFooter() {
  const pathname = usePathname();
  if (pathname.startsWith('/dashboard') || pathname.startsWith('/admin')) return null;
  return (
    <footer className="public-footer">
      <div className="public-footer-top">
        <Brand />
        <p>Kite는 Kubernetes 기반으로 가상머신을 할당하는 작은 클라우드 서비스입니다. 소규모 팀이 한 대의 온프레미스 컴퓨터에서, 단일 노드 환경의 격리된 VM 작업 공간을 운영하도록 설계했습니다.</p>
        <a href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank" rel="noreferrer" className="public-footer-repository button button--ink"><GithubOutlined aria-hidden="true" /><span>Github</span></a>
      </div>
      <div className="public-footer-bottom">
        <span>© {new Date().getUTCFullYear()} Kite. Created by <a className="public-footer-credit" href="https://github.com/hy3ons" target="_blank" rel="noreferrer">Hy3ons</a>.</span>
        <nav aria-label="이용 안내 및 정책"><Link href="/kite-docs">이용 가이드</Link><Link href="/privacy">개인정보처리방침</Link><Link href="/terms">서비스 이용약관</Link></nav>
        <span>Open source. On your terms.</span>
      </div>
    </footer>
  );
}
