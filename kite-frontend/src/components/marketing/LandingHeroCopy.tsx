import Link from 'next/link';
import { ArrowRightOutlined } from '@ant-design/icons';

// LandingHeroCopy shares the homepage introduction with its inert animation scene.
// preview scopes the heading ID and disables link prefetch in the miniature page.
// startHref carries the server-selected destination; inert previews default to login.
export function LandingHeroCopy({ preview = false, startHref = '/login' }: { readonly preview?: boolean; readonly startHref?: '/login' | '/dashboard' }) {
  return <div className="marketing-hero-copy">
    <p className="marketing-eyebrow">Your own space in the cloud</p>
    <h1 id={preview ? 'demo-marketing-title' : 'marketing-title'}>당신의 클라우드,<br />가볍게 펼치세요.</h1>
    <p className="marketing-lead">가상머신을 만들고, 연결하고, 관리하는 일.<br className="marketing-desktop-break" /> Kubernetes 위에서 Kite로 간결하게.</p>
    <div className="marketing-actions">
      <Link href={startHref} prefetch={preview ? false : null} className="marketing-button marketing-button--primary">Kite 시작하기 <ArrowRightOutlined /></Link>
      <Link href="/kite-docs" prefetch={preview ? false : null} className="marketing-text-link">어떻게 동작하나요? <ArrowRightOutlined /></Link>
    </div>
  </div>;
}
