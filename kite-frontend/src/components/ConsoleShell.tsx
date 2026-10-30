'use client';

import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { flushSync } from 'react-dom';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Button, Tooltip } from 'antd';
import { AppstoreOutlined, CloudServerOutlined, GiftOutlined, TeamOutlined, SettingOutlined, BookOutlined, GithubOutlined, MenuOutlined, CloseOutlined, RightOutlined, MenuFoldOutlined, MenuUnfoldOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { useConsoleNavigation } from '../store/useConsoleNavigation';
import { USE_MOCK_API } from '../config/debug';
import { Brand } from './Brand';
import { AccountMenu } from './AccountMenu';

type ConsoleShellProps = {
  readonly children: ReactNode;
  readonly title?: string;
  readonly eyebrow?: string;
  readonly actions?: ReactNode;
  readonly section?: 'workspace' | 'admin' | 'settings';
  readonly preview?: boolean;
};

// ConsoleShell gives authenticated screens one navigation, identity, and page frame.
// section chooses the current role-aware navigation group; actions belongs to this page.
export function ConsoleShell({ children, title, eyebrow, actions, section = 'workspace', preview = false }: ConsoleShellProps) {
  const { accessLevel } = useAuthStore();
  const currentPathname = usePathname();
  const pathname = preview ? '/dashboard' : currentPathname;
  const [menuOpen, setMenuOpen] = useState(false);
  const { collapsed, toggleCollapsed } = useConsoleNavigation();
  const sidebarTransition = useRef<ViewTransition | null>(null);
  const [anchor, setAnchor] = useState('');
  useEffect(() => {
    const updateAnchor = () => setAnchor(preview ? '' : window.location.hash);
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
  }, [preview]);

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
  const Content = preview ? 'div' : 'main';
  const navigationId = preview ? 'preview-console-navigation' : 'console-navigation';
  const isVm = pathname.startsWith('/dashboard/kite-machine/');
  const groupLabel = section === 'settings' ? '설정' : section === 'admin' ? '관리' : '워크스페이스';
  const links = [
    { href: '/dashboard', hash: '', label: '개요', icon: <AppstoreOutlined />, active: section === 'workspace' && !isVm && !anchor },
    { href: '/dashboard#instances', hash: '#instances', label: '가상머신', icon: <CloudServerOutlined />, active: isVm || (section === 'workspace' && anchor === '#instances') },
    { href: '/dashboard#offers', hash: '#offers', label: '할당받은 VM', icon: <GiftOutlined />, active: section === 'workspace' && !isVm && anchor === '#offers' },
  ];
  return (
    <div className={`console-shell${collapsed ? ' is-sidebar-collapsed' : ''}${preview ? ' console-shell--preview' : ''}`}>
      <header className="console-topbar">
        <Button type="text" icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} aria-label={collapsed ? '사이드바 펼치기' : '사이드바 접기'} aria-expanded={!collapsed} aria-controls={navigationId} onClick={toggleSidebar} className="console-sidebar-toggle appbar-action" />
        <Button type="text" icon={menuOpen ? <CloseOutlined /> : <MenuOutlined />} aria-label={menuOpen ? '탐색 메뉴 닫기' : '탐색 메뉴 열기'} aria-expanded={menuOpen} aria-controls={navigationId} onClick={() => setMenuOpen(!menuOpen)} className="console-menu-toggle appbar-action" />
        <Brand href={preview ? '/' : '/dashboard'} className="console-brand" prefetch={preview ? false : undefined} />
        <span className="console-product-name">Cloud console</span>
        <div className="console-topbar-end">
          {USE_MOCK_API && <span className="environment-label"><span />Mock</span>}
          <Tooltip title="이용 가이드"><Link className="icon-link appbar-action" href="/kite-docs" prefetch={preview ? false : undefined} aria-label="이용 가이드"><BookOutlined /></Link></Tooltip>
          <AccountMenu preview={preview} />
        </div>
      </header>
      {menuOpen && <button className="console-nav-scrim" aria-label="탐색 메뉴 닫기" onClick={() => setMenuOpen(false)} />}
      <aside id={navigationId} className={`console-sidebar${menuOpen ? ' is-open' : ''}`}>
        <div className="sidebar-workspace"><span className="sidebar-workspace-icon"><CloudServerOutlined /></span><span>내 클라우드<small>개인 워크스페이스</small></span></div>
        <nav aria-label="콘솔 탐색">
          <p className="sidebar-label">워크스페이스</p>
          {links.map((link) => <Link key={link.href} href={link.href} prefetch={preview ? false : undefined} aria-current={link.active ? 'page' : undefined} className={`sidebar-link${link.active ? ' is-active' : ''}`} onClick={() => { setMenuOpen(false); setAnchor(link.hash); }} title={link.label}>{link.icon}<span>{link.label}</span></Link>)}
          {level >= 2 && <><p className="sidebar-label sidebar-label--section">관리</p><Link href="/admin/dashboard" aria-current={section === 'admin' ? 'page' : undefined} className={`sidebar-link${section === 'admin' ? ' is-active' : ''}`} onClick={() => setMenuOpen(false)} title="사용자 및 리소스"><TeamOutlined /><span>사용자 및 리소스</span></Link></>}
          {level >= 3 && <Link href="/admin/settings" aria-current={section === 'settings' ? 'page' : undefined} className={`sidebar-link${section === 'settings' ? ' is-active' : ''}`} onClick={() => setMenuOpen(false)} title="시스템 설정"><SettingOutlined /><span>시스템 설정</span></Link>}
        </nav>
        <div className="sidebar-bottom"><Link href="/kite-docs" prefetch={preview ? false : undefined} className="sidebar-link" title="문서 및 도움말"><BookOutlined /><span>문서 및 도움말</span></Link><a href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank" rel="noreferrer" className="sidebar-link" title="GitHub"><GithubOutlined /><span>GitHub</span></a><div className="sidebar-version">Kite / Open source cloud</div></div>
      </aside>
      <Content id={preview ? undefined : 'main-content'} className="console-main" aria-label={title ?? groupLabel}>
        <div className="console-breadcrumb"><Link href="/dashboard" prefetch={preview ? false : undefined}>Kite</Link><RightOutlined /><span>{eyebrow ?? groupLabel}</span></div>
        {(title || actions) && <div className="page-heading">{title && <div><h1>{title}</h1></div>}{actions && <div className="page-heading-actions">{actions}</div>}</div>}
        {children}
      </Content>
    </div>
  );
}
