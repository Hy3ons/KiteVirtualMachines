'use client';

import { useRef } from 'react';
import { Button, ConfigProvider } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { AuthProvider } from '../../store/useAuthStore';
import { ConsoleNavigationProvider } from '../../store/useConsoleNavigation';
import { ConsoleShell } from '../ConsoleShell';
import { StatusBadge } from '../StatusBadge';
import { DemoLogin } from './demo/DemoLogin';
import { DemoLanding } from './demo/DemoLanding';
import { DemoDashboard } from './demo/DemoDashboard';
import { DemoConsole } from './demo/DemoConsole';
import { DemoControls } from './demo/DemoControls';
import { DemoBrowserChrome } from './demo/DemoBrowserChrome';
import { DEMO_HEIGHT, DEMO_WIDTH } from './demo/demoStory';
import { DEMO_USER, DEMO_VM, demoIgnore } from './demo/demoData';
import { useDemoPlayback } from './demo/useDemoPlayback';

// ConsolePreview embeds the real workspace primitives in an isolated, inert product demonstration.
// The logical canvas preserves the live 900px content width; GSAP never clicks or calls real APIs.
export function ConsolePreview() {
  const root = useRef<HTMLElement>(null);
  const playback = useDemoPlayback(root);
  const { snapshot } = playback;
  return (
    <figure ref={root} className={`product-demo${playback.reduced ? ' is-reduced-motion' : ''}`} aria-label="Kite 로그인과 워크스페이스 미리보기">
      <DemoControls playback={playback} />
      <div className="demo-window">
        <DemoBrowserChrome scene={snapshot.scene} />
        <div className="demo-viewport" data-demo-viewport style={{ aspectRatio: `${DEMO_WIDTH} / ${DEMO_HEIGHT}` }}>
          <div className="demo-canvas" data-demo-canvas style={{ width: DEMO_WIDTH, height: DEMO_HEIGHT, transform: `scale(${playback.scale})` }} aria-hidden="true" inert>
            <div className="demo-camera" data-demo-camera>
            <ConfigProvider getPopupContainer={() => root.current?.querySelector<HTMLElement>('[data-demo-camera]') ?? document.body}>
              <AuthProvider user={DEMO_USER}><ConsoleNavigationProvider>
                {snapshot.scene === 'landing' ? <DemoLanding /> : snapshot.scene === 'login' ? <DemoLogin snapshot={snapshot} /> : <ConsoleShell preview section="workspace" {...(snapshot.scene === 'console' ? { title: DEMO_VM.name, eyebrow: '가상머신 / 콘솔' } : {})} actions={snapshot.scene === 'console' ? <><span className="serial-page-status"><StatusBadge phase="Running" /></span><Button icon={<ArrowLeftOutlined />} onClick={demoIgnore}>VM 상세</Button><Button icon={<ReloadOutlined />} onClick={demoIgnore}>새로고침</Button></> : undefined}>
                  {snapshot.scene === 'console' ? <DemoConsole snapshot={snapshot} /> : <DemoDashboard snapshot={snapshot} />}
                </ConsoleShell>}
              </ConsoleNavigationProvider></AuthProvider>
            </ConfigProvider>
            </div>
            <span className="demo-pointer" data-demo-pointer>
              <span className="demo-pointer-appearance" style={{ transform: `scale(${1 / playback.scale})` }}>
                <span className="demo-click-ring" data-demo-click />
                <svg className="demo-pointer-glyph" width="28" height="35" viewBox="0 0 32 40" fill="none"><path d="M0 0V29L8.5 22.3L15 36L21.2 33.2L14.8 19.7H26.8L0 0Z" fill="var(--ink)" stroke="var(--canvas)" strokeWidth="2.4" strokeLinejoin="round" /></svg>
              </span>
            </span>
          </div>
        </div>
        <div className="demo-progress" aria-hidden="true"><span data-demo-progress /></div>
      </div>
    </figure>
  );
}
