import { AppstoreOutlined, BookOutlined, CopyOutlined, DownOutlined, LeftOutlined, PlusOutlined } from '@ant-design/icons';
import type { DemoScene } from './demoStory';

// DemoBrowserChrome frames the local animation with Safari-style compact tabs and inert navigation.
// scene selects the sample address and tab title; none of these controls navigate.
export function DemoBrowserChrome({ scene }: { readonly scene: DemoScene }) {
  const consoleOpen = scene === 'console';
  const path = scene === 'landing' ? '/' : scene === 'login' ? '/login' : consoleOpen ? '/dashboard/kite-machine/demo-vm/console' : '/dashboard';
  const title = scene === 'landing' ? 'Kite | 나만의 클라우드' : scene === 'login' ? '로그인 | Kite' : consoleOpen ? 'demo-vm · 콘솔 | Kite' : '워크스페이스 | Kite';
  return <div className="demo-browser-chrome" aria-hidden="true">
    <div className="demo-browser-dots"><i /><i /><i /></div>
    <div className="demo-browser-navigation">
      <span className="demo-browser-sidebar demo-browser-capsule"><svg width="16" height="16" viewBox="0 0 20 20" fill="none"><rect x="2" y="3" width="16" height="14" rx="2.5" /><path d="M7 3v14M4.5 6h.1M4.5 9h.1" /></svg><DownOutlined /></span>
      <span className="demo-browser-back demo-browser-capsule"><LeftOutlined /></span>
    </div>
    <div className="demo-browser-tabs">
      <div className="demo-browser-tab"><BookOutlined /><span>사용 가이드 | Kite</span></div>
      <div className="demo-browser-tab"><AppstoreOutlined /><span>워크스페이스 | Kite</span></div>
      <div className="demo-browser-address" title={`${title} · kite.example${path}`}><img src="/favicon.svg" alt="" width="13" height="13" /><span>kite.example</span><svg className="demo-browser-reader" width="13" height="14" viewBox="0 0 18 20" fill="none"><rect x="3" y="2" width="12" height="8" rx="2" /><path d="M3 14h12M3 18h9" /></svg></div>
    </div>
    <div className="demo-browser-tools demo-browser-capsule">
      <svg className="demo-browser-share" width="16" height="17" viewBox="0 0 20 22" fill="none"><path d="M10 14V2M6 6l4-4 4 4M6 9H4a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8a2 2 0 0 0-2-2h-2" /></svg><PlusOutlined /><CopyOutlined />
    </div>
  </div>;
}
