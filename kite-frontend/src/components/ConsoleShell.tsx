'use client';

import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { flushSync } from 'react-dom';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Button, Tooltip } from 'antd';
import { AppstoreOutlined, CloudServerOutlined, GiftOutlined, TeamOutlined, SettingOutlined, BookOutlined, GithubOutlined, LogoutOutlined, MenuOutlined, CloseOutlined, RightOutlined, UserOutlined, MenuFoldOutlined, MenuUnfoldOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { useLogout } from '../hooks/useLogout';
import { useConsoleNavigation } from '../store/useConsoleNavigation';
import { USE_MOCK_API } from '../config/debug';
import { Brand } from './Brand';

type ConsoleShellProps = {
  readonly children: ReactNode;
  readonly title?: string;
  readonly eyebrow?: string;
  readonly actions?: ReactNode;
  readonly section?: 'workspace' | 'admin' | 'settings';
};

// ConsoleShell gives authenticated screens one navigation, identity, and page frame.
// section chooses the current role-aware navigation group; actions belongs to this page.
export function ConsoleShell({ children, title, eyebrow, actions, section = 'workspace' }: ConsoleShellProps) {
  const { username, email, accessLevel } = useAuthStore();
  const logout = useLogout();
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);
  const { collapsed, toggleCollapsed } = useConsoleNavigation();
  const sidebarTransition = useRef<ViewTransition | null>(null);
  const [anchor, setAnchor] = useState('');
  useEffect(() => {
    const updateAnchor = () => setAnchor(window.location.hash);
    const closeOnEscape = (event: KeyboardEvent) => { if (event.key === 'Escape') setMenuOpen(false); };
    queueMicrotask(updateAnchor);
    window.addEventListener('hashchange', updateAnchor);
    window.addEventListener('popstate', updateAnchor);
    window.addEventListener('keydown', closeOnEscape);
    return () => {
      sidebarTransition.current?.skipTransition();
      window.removeEventListener('hashchange', updateAnchor);
      window.removeEventListener('popstate', updateAnchor);
      window.removeEventListener('keydown', closeOnEscape);
    };
  }, []);

  // toggleSidebar snapshots the desktop layout before changing its rail width.
  // Reduced motion and older browsers switch directly; the mobile drawer stays independent.
  const toggleSidebar = () => {
    sidebarTransition.current?.skipTransition();
    const update = () => flushSync(toggleCollapsed);
    if (!document.startViewTransition || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      update();
      return;
    }
    sidebarTransition.current = document.startViewTransition(update);
  };
  const level = accessLevel ?? 0;
  const isVm = pathname.startsWith('/dashboard/kite-machine/');
  const groupLabel = section === 'settings' ? '설정' : section === 'admin' ? '관리' : '워크스페이스';
  const links = [
    { href: '/dashboard', hash: '', label: '개요', icon: <AppstoreOutlined />, active: section === 'workspace' && !isVm && !anchor },
    { href: '/dashboard#instances', hash: '#instances', label: '가상머신', icon: <CloudServerOutlined />, active: isVm || (section === 'workspace' && anchor === '#instances') },
    { href: '/dashboard#offers', hash: '#offers', label: '할당받은 VM', icon: <GiftOutlined />, active: section === 'workspace' && !isVm && anchor === '#offers' },
  ];
  return (
    <div className={`console-shell${collapsed ? ' is-sidebar-collapsed' : ''}`}>
      <header className="console-topbar">
        <Button type="text" icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} aria-label={collapsed ? '사이드바 펼치기' : '사이드바 접기'} aria-expanded={!collapsed} aria-controls="console-navigation" onClick={toggleSidebar} className="console-sidebar-toggle" />
        <Button type="text" icon={menuOpen ? <CloseOutlined /> : <MenuOutlined />} aria-label={menuOpen ? '탐색 메뉴 닫기' : '탐색 메뉴 열기'} aria-expanded={menuOpen} aria-controls="console-navigation" onClick={() => setMenuOpen(!menuOpen)} className="console-menu-toggle" />
        <Brand href="/dashboard" className="console-brand" />
        <span className="console-product-name">Cloud console</span>
        <div className="console-topbar-end">
          {USE_MOCK_API && <span className="environment-label"><span />Mock</span>}
          <Tooltip title="이용 가이드"><Link className="icon-link" href="/kite-docs" aria-label="이용 가이드"><BookOutlined /></Link></Tooltip>
          <div className="console-account" title={[username, email].filter(Boolean).join('\n')} aria-label={[username, email].filter(Boolean).join(', ')}><span className="account-avatar" aria-hidden="true"><UserOutlined /></span><span className="console-account-name"><span>{username}</span><small>{email}</small></span></div>
          <Tooltip title="로그아웃"><Button type="text" icon={<LogoutOutlined />} aria-label="로그아웃" onClick={logout} /></Tooltip>
        </div>
      </header>
      {menuOpen && <button className="console-nav-scrim" aria-label="탐색 메뉴 닫기" onClick={() => setMenuOpen(false)} />}
      <aside id="console-navigation" className={`console-sidebar${menuOpen ? ' is-open' : ''}`}>
        <div className="sidebar-workspace"><span className="sidebar-workspace-icon"><CloudServerOutlined /></span><span>내 클라우드<small>개인 워크스페이스</small></span></div>
        <nav aria-label="콘솔 탐색">
          <p className="sidebar-label">워크스페이스</p>
          {links.map((link) => <Link key={link.href} href={link.href} aria-current={link.active ? 'page' : undefined} className={`sidebar-link${link.active ? ' is-active' : ''}`} onClick={() => { setMenuOpen(false); setAnchor(link.hash); }} title={link.label}>{link.icon}<span>{link.label}</span></Link>)}
          {level >= 2 && <><p className="sidebar-label sidebar-label--section">관리</p><Link href="/admin/dashboard" aria-current={section === 'admin' ? 'page' : undefined} className={`sidebar-link${section === 'admin' ? ' is-active' : ''}`} onClick={() => setMenuOpen(false)} title="사용자 및 리소스"><TeamOutlined /><span>사용자 및 리소스</span></Link></>}
          {level >= 3 && <Link href="/admin/settings" aria-current={section === 'settings' ? 'page' : undefined} className={`sidebar-link${section === 'settings' ? ' is-active' : ''}`} onClick={() => setMenuOpen(false)} title="시스템 설정"><SettingOutlined /><span>시스템 설정</span></Link>}
        </nav>
        <div className="sidebar-bottom"><Link href="/kite-docs" className="sidebar-link" title="문서 및 도움말"><BookOutlined /><span>문서 및 도움말</span></Link><a href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank" rel="noreferrer" className="sidebar-link" title="GitHub"><GithubOutlined /><span>GitHub</span></a><div className="sidebar-version">Kite / Open source cloud</div></div>
      </aside>
      <main id="main-content" className="console-main" aria-label={title ?? groupLabel}>
        <div className="console-breadcrumb"><Link href="/dashboard">Kite</Link><RightOutlined /><span>{eyebrow ?? groupLabel}</span></div>
        {(title || actions) && <div className="page-heading">{title && <div><h1>{title}</h1></div>}{actions && <div className="page-heading-actions">{actions}</div>}</div>}
        {children}
      </main>
    </div>
  );
}
