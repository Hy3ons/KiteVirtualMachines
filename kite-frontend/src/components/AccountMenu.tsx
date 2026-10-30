'use client';

import { useId, useRef, useState } from 'react';
import type { KeyboardEvent } from 'react';
import Link from 'next/link';
import { Popover } from 'antd';
import { UserOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { canManageSystem } from '../domain/access';
import { useLogout } from '../hooks/useLogout';
import { AccountVmUsage } from './AccountVmUsage';

// AccountMenu shares a click-open identity panel across public and console appbars.
// preview disables opening in the inert marketing canvas, which must never request account data.
// A live content shell stays defined while closed so Ant Design can emit the first open request.
export function AccountMenu({ preview = false }: { readonly preview?: boolean }) {
  const { username, email, accessLevel } = useAuthStore();
  const logout = useLogout();
  const [open, setOpen] = useState(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const panel = useRef<HTMLDivElement>(null);
  const id = useId();

  // closeOnEscape returns keyboard focus to the person button without scrolling the page.
  const closeOnEscape = (event: KeyboardEvent<HTMLElement>) => {
    if (event.key !== 'Escape') return;
    event.stopPropagation();
    setOpen(false);
    trigger.current?.focus({ preventScroll: true });
  };

  return <Popover trigger={preview ? [] : ['click']} placement="bottomRight" arrow={false} open={!preview && open} onOpenChange={(next) => { if (!preview) setOpen(next); }} destroyOnHidden classNames={{ root: 'account-popover' }} afterOpenChange={(next) => { if (next) panel.current?.focus({ preventScroll: true }); }} content={!preview ? <div ref={panel} id={id} className="account-panel" role="dialog" aria-label="내 계정" tabIndex={-1} onKeyDown={closeOnEscape}>
    <div className="account-panel-identity"><span className="account-avatar" aria-hidden="true"><UserOutlined /></span><div><strong>{username}</strong><span>{email}</span></div></div>
    {open && <AccountVmUsage />}
    <div className="account-panel-actions"><Link href="/dashboard" prefetch={false} className="appbar-action" onClick={() => setOpen(false)}>내 대시보드</Link>{canManageSystem(accessLevel) && <Link href="/admin/settings" prefetch={false} className="appbar-action" onClick={() => setOpen(false)}>시스템 설정</Link>}<button type="button" className="appbar-action account-logout" onClick={() => { setOpen(false); void logout(); }}>로그아웃</button></div>
  </div> : null}>
    <button ref={trigger} type="button" className="appbar-action account-menu-trigger" aria-label="내 계정" aria-describedby={`${id}-summary`} aria-haspopup="dialog" aria-expanded={!preview && open} aria-controls={open ? id : undefined} aria-disabled={preview || undefined} onKeyDown={closeOnEscape}>
      <UserOutlined aria-hidden="true" />
      <span id={`${id}-summary`} className="account-menu-summary"><strong title={username ?? undefined}>{username}</strong><span title={email ?? undefined}>{email}</span></span>
    </button>
  </Popover>;
}
