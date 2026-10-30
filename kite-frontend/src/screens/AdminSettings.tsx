'use client';

import type { ConfigResponse } from '../api/types';
import { ConsoleShell } from '../components/ConsoleShell';
import { useAdminSettings } from './admin/useAdminSettings';
import { RoutingSettings } from './admin/RoutingSettings';
import { GatewaySettings } from './admin/GatewaySettings';
import { ContactSettings } from './admin/ContactSettings';
import { SecuritySettings } from './admin/SecuritySettings';

const sections = [
  { id: 'domain', label: '도메인' },
  { id: 'https', label: 'HTTPS 정책' },
  { id: 'gateway', label: 'SSH 게이트웨이' },
  { id: 'contact', label: '관리자 연락처' },
  { id: 'runtime', label: '런타임 보안' },
  { id: 'certificates', label: 'TLS 인증서' },
] as const;

// AdminSettings renders anchored configuration groups; initialData supplies verified server settings.
export function AdminSettings({ initialData = null }: { readonly initialData?: ConfigResponse | null }) {
  const settings = useAdminSettings(initialData);
  return (
    <ConsoleShell section="settings" eyebrow="플랫폼 관리" title="시스템 설정">
      <div className="admin-settings-layout">
        <nav className="admin-settings-nav" aria-label="시스템 설정 항목">
          <span className="admin-nav-label">설정 항목</span>
          {sections.map((section) => <a key={section.id} href={`#${section.id}`}>{section.label}</a>)}
        </nav>
        <div className="admin-settings-sections">
          <RoutingSettings settings={settings} initialData={initialData} />
          <GatewaySettings settings={settings} />
          <ContactSettings settings={settings} initialData={initialData} />
          <SecuritySettings settings={settings} />
        </div>
      </div>
    </ConsoleShell>
  );
}
