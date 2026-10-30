import Link from 'next/link';
import { Brand } from '../components/Brand';
import { createPageMetadata } from '../lib/metadata';

// generateMetadata prevents unknown URLs from inheriting the homepage canonical identity.
export function generateMetadata() {
  return createPageMetadata('페이지를 찾을 수 없습니다', { noIndex: true, description: '요청한 페이지를 찾을 수 없습니다. 주소를 확인하거나 Kite 홈페이지로 돌아가세요.' });
}

// NotFound offers a useful return path for unknown App Router URLs.
export default function NotFound() {
  return <main id="main-content" className="route-message"><Brand /><p>404 / Page not found</p><h1>여기에는 아직<br />작업 공간이 없어요.</h1><p>주소를 다시 확인하거나 콘솔에서 가상머신을 찾아보세요.</p><div className="page-heading-actions"><Link className="button button--ink" href="/dashboard">콘솔 열기</Link><Link className="button button--soft" href="/">홈으로</Link></div></main>;
}
