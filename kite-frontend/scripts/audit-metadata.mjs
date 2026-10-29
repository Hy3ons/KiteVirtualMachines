import assert from 'node:assert/strict';

const origin = new URL(process.argv[2] || 'http://127.0.0.1:5173').origin;
const publicOrigin = new URL(process.argv[3] || origin).origin;
const crawlerHeaders = { 'user-agent': 'facebookexternalhit/1.1' };
const imagePaths = new Set();
const routes = [
  ['/', 'Kite | Kubernetes 기반 VM 플랫폼', false],
  ['/kite-docs', '사용 가이드 | Kite', false],
  ['/kite-docs?utm_source=metadata-audit', '사용 가이드 | Kite', false],
  ['/login', '로그인 | Kite', true],
  ['/login?sessionExpired=true', '로그인 | Kite', true],
  ['/signup', '회원가입 | Kite', true],
  ['/privacy', '개인정보처리방침 | Kite', true],
  ['/terms', '서비스 이용약관 | Kite', true],
  ['/dashboard', '워크스페이스 | Kite', true],
  ['/admin/dashboard', '사용자 및 리소스 | Kite', true],
  ['/admin/settings', '시스템 설정 | Kite', true],
  ['/dashboard/kite-machine/dev-vm-1', 'dev-vm-1 | 가상머신 | Kite', true],
  ['/dashboard/kite-machine/dev-vm-1/console', 'dev-vm-1 | 웹 콘솔 | Kite', true],
  ['/metadata-audit-missing-page', '페이지를 찾을 수 없습니다 | Kite', true],
];

// attributes reads the quoted SSR attributes used by Next metadata tags.
// tag is an opening HTML tag; the result supports semantic checks independent of order.
function attributes(tag) {
  return Object.fromEntries([...tag.matchAll(/([\w:-]+)="([^"]*)"/g)].map((match) => [match[1], match[2]]));
}

// metaValues selects named/property metadata from the crawler's completed head.
// head is isolated from SVG titles and RSC payloads so document checks remain precise.
function metaValues(head, key) {
  return [...head.matchAll(/<meta\b[^>]*>/g)].map((match) => attributes(match[0]))
    .filter((value) => value.name === key || value.property === key).map((value) => value.content);
}

// request only reads HTTP responses; no browser, UI scripts, login or mutations run.
function request(path, headers = crawlerHeaders) {
  return fetch(new URL(path, origin), { headers, redirect: 'manual', signal: AbortSignal.timeout(15000) });
}

for (const [path, title, noIndex] of routes) {
  const response = await request(path);
  if (response.status === 307 && (path.startsWith('/dashboard') || path.startsWith('/admin'))) {
    assert.ok(response.headers.get('location')?.startsWith('/login?requireLogin='), path);
    console.log(`PASS ${path}: anonymous access redirects to login`);
    continue;
  }
  assert.equal(response.status, path.includes('missing-page') ? 404 : 200, path);
  const html = await response.text();
  const head = html.match(/<head>([\s\S]*?)<\/head>/)?.[1];
  assert.ok(head, `${path}: crawler receives a complete head`);
  assert.deepEqual([...head.matchAll(/<title>([^<]*)<\/title>/g)].map((match) => match[1]), [title], path);
  for (const key of ['og:title', 'twitter:title']) assert.deepEqual(metaValues(head, key), [title], `${path}: ${key}`);
  const description = metaValues(head, 'description');
  assert.equal(description.length, 1, path);
  assert.ok(description[0].length >= 10, path);
  for (const key of ['og:description', 'twitter:description']) assert.deepEqual(metaValues(head, key), description, `${path}: ${key}`);
  assert.deepEqual(metaValues(head, 'og:site_name'), ['Kite'], path);
  assert.deepEqual(metaValues(head, 'og:locale'), ['ko_KR'], path);
  assert.deepEqual(metaValues(head, 'og:type'), ['website'], path);
  assert.deepEqual(metaValues(head, 'twitter:card'), ['summary_large_image'], path);
  assert.ok(metaValues(head, 'robots').some((value) => noIndex ? value.includes('noindex') : value.includes('index') && !value.includes('noindex')), path);
  const canonical = [...head.matchAll(/<link\b[^>]*>/g)].map((match) => attributes(match[0])).filter((value) => value.rel === 'canonical');
  assert.equal(canonical.length, noIndex ? 0 : 1, `${path}: canonical indexing policy`);
  const pathname = new URL(path, origin).pathname;
  if (!noIndex) assert.equal(new URL(canonical[0].href).href, new URL(pathname, publicOrigin).href, path);
  if (!path.includes('missing-page')) {
    assert.equal(new URL(metaValues(head, 'og:url')[0]).href, new URL(pathname, publicOrigin).href, path);
  } else {
    assert.equal(metaValues(head, 'og:url').length, 0, '404 must not claim the homepage identity');
  }
  assert.deepEqual(metaValues(head, 'og:image:width'), ['1200'], path);
  assert.deepEqual(metaValues(head, 'og:image:height'), ['630'], path);
  assert.deepEqual(metaValues(head, 'og:image:type'), ['image/jpeg'], path);
  assert.ok(metaValues(head, 'og:image:alt')[0]?.length > 10, path);
  assert.deepEqual(metaValues(head, 'twitter:image:alt'), metaValues(head, 'og:image:alt'), path);
  const images = metaValues(head, 'og:image');
  assert.equal(images.length, 1, path);
  assert.deepEqual(metaValues(head, 'twitter:image'), images, path);
  const image = new URL(images[0]);
  assert.equal(image.origin, publicOrigin, `${path}: absolute image origin`);
  assert.equal(image.pathname, pathname === '/kite-docs' ? '/images/social/kite-guide-social.jpg' : '/images/social/kite-social.jpg', path);
  imagePaths.add(image.pathname);
  console.log(`PASS ${path}: title, descriptions, OG/Twitter, image, canonical and robots`);
}

for (const path of imagePaths) {
  const response = await request(path);
  assert.equal(response.status, 200, path);
  assert.ok(response.headers.get('content-type')?.startsWith('image/jpeg'), path);
  const bytes = new Uint8Array(await response.arrayBuffer());
  assert.ok(bytes.length > 1000 && bytes.length < 300000, `${path}: useful optimized card`);
  assert.deepEqual([...bytes.slice(0, 3)], [255, 216, 255], `${path}: JPEG signature`);
  console.log(`PASS ${path}: public JPEG response (${bytes.length} bytes)`);
}
const twitterResponse = await request('/', { 'user-agent': 'Twitterbot/1.0' });
const twitterHead = (await twitterResponse.text()).match(/<head>([\s\S]*?)<\/head>/)?.[1] || '';
assert.deepEqual(metaValues(twitterHead, 'twitter:card'), ['summary_large_image'], 'Twitter crawler head');
const robots = await request('/robots.txt');
const robotsText = await robots.text();
assert.equal(robots.status, 200);
assert.ok(robotsText.includes(`Sitemap: ${publicOrigin}/sitemap.xml`));
for (const path of ['/api/', '/dashboard', '/admin']) assert.ok(robotsText.includes(`Disallow: ${path}`));
const sitemap = await request('/sitemap.xml');
const xml = await sitemap.text();
assert.equal(sitemap.status, 200);
assert.deepEqual([...xml.matchAll(/<loc>([^<]+)<\/loc>/g)].map((match) => new URL(match[1]).href).sort(), [new URL('/', publicOrigin).href, new URL('/kite-docs', publicOrigin).href].sort());
console.log('PASS Twitter crawler, robots.txt and sitemap.xml; HTTP-only audit complete');
