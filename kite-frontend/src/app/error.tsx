'use client';

import Link from 'next/link';
import { Brand } from '../components/Brand';

// ErrorPage retries failed SSR/API loads without treating an outage as logout.
// reset asks Next.js to render the failed route segment again.
export default function ErrorPage({ reset }: { readonly reset: () => void }) {
  return <main id="main-content" className="route-message"><Brand /><h1>잠시 연결이 끊겼습니다.</h1><p>페이지를 불러오지 못했습니다. API 연결 상태를 확인한 뒤 다시 시도해 주세요.</p><div className="page-heading-actions"><button className="button button--ink" onClick={reset}>다시 시도</button><Link className="button button--soft" href="/">홈으로</Link></div></main>;
}
