'use client';

import type { ReactNode } from 'react';
import Link from 'next/link';
import { ArrowUpOutlined, GithubOutlined } from '@ant-design/icons';
import { Brand } from './Brand';

// GlobalHeader supplies public navigation; rightContent optionally replaces account links.
// Console pages use ConsoleShell to keep marketing and task navigation distinct.
export function GlobalHeader({ rightContent }: { readonly rightContent?: ReactNode }) {
  return (
    <header className="public-header">
      <Brand />
      <nav className="public-nav" aria-label="주 메뉴">
        <Link href="/#platform">플랫폼</Link>
        <Link href="/kite-docs">가이드</Link>
        <a href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank" rel="noreferrer" className="public-repository"><GithubOutlined /><span>GitHub</span></a>
      </nav>
      <div className="public-header-actions">{rightContent ?? <><Link href="/login" className="public-login-link">로그인</Link><Link href="/dashboard" className="button button--ink button--compact">콘솔 열기 <ArrowUpOutlined className="diagonal-arrow" /></Link></>}</div>
    </header>
  );
}
