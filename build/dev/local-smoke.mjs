import assert from 'node:assert/strict';
import { randomBytes } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';

const apiOrigin = 'http://127.0.0.1:8080';
const frontendOrigin = process.env.FRONTEND_ORIGIN;
const credentialsPath = new URL('../../.local-dev/login.json', import.meta.url);

// request sends a bounded local HTTP request and checks its expected status.
// path and options select an API call; status defaults to 200 for smoke checks.
async function request(path, options = {}, status = 200) {
  const response = await fetch(`${apiOrigin}/api/v1${path}`, {
    ...options, signal: AbortSignal.timeout(10_000),
  });
  assert.equal(response.status, status, `${path}: ${await response.clone().text()}`);
  return response;
}

assert.equal((await (await request('/health')).json()).status, 'ok');
await request('/me', {}, 401);

let credentials;
try {
  credentials = JSON.parse(await readFile(credentialsPath, 'utf8'));
} catch (error) {
  if (error.code !== 'ENOENT') throw error;
  credentials = { username: 'local-admin', email: 'local-admin@kite.local', password: randomBytes(24).toString('base64url') };
  await request('/auth/signup', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(credentials),
  }, 201);
  await mkdir(new URL('../../.local-dev/', import.meta.url), { recursive: true, mode: 0o700 });
  await writeFile(credentialsPath, `${JSON.stringify(credentials, null, 2)}\n`, { mode: 0o600 });
}

await request('/auth/login', {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email: credentials.email, password: 'incorrect-password' }),
}, 401);
const login = await request('/auth/login', {
  method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(credentials),
});
const setCookie = login.headers.get('set-cookie');
assert.match(setCookie ?? '', /accessToken=/);
assert.match(setCookie, /HttpOnly/i);
const cookie = setCookie.split(';')[0];
const headers = { Cookie: cookie };
const me = await (await request('/me', { headers })).json();
assert.equal(me.user.username, credentials.username);
assert.equal(me.user.access_level, 3, 'Use a fresh kite-dev cluster: the smoke account should be its first admin.');
for (const endpoint of ['/vms', '/vm-offers', '/config', '/admin/users', '/admin/vms', '/admin/settings']) {
  await request(endpoint, { headers });
}

if (frontendOrigin) {
  const origin = new URL(frontendOrigin);
  assert.ok(['127.0.0.1', 'localhost'].includes(origin.hostname), 'SSR smoke must target a local stage server.');
  const proxiedLogin = await fetch(`${origin.origin}/api/v1/auth/login`, {
    method: 'POST', headers: { 'Content-Type': 'application/json', Origin: origin.origin },
    body: JSON.stringify(credentials), signal: AbortSignal.timeout(10_000),
  });
  assert.equal(proxiedLogin.status, 200);
  assert.match(proxiedLogin.headers.get('set-cookie') ?? '', /HttpOnly/i);
  const stageHeaders = { Cookie: proxiedLogin.headers.get('set-cookie').split(';')[0] };
  const anonymous = await fetch(`${origin.origin}/dashboard`, { redirect: 'manual', signal: AbortSignal.timeout(60_000) });
  assert.equal(anonymous.status, 307);
  assert.match(anonymous.headers.get('location') ?? '', /requireLogin/);
  for (const path of ['/dashboard', '/admin/dashboard', '/admin/settings']) {
    const response = await fetch(`${origin.origin}${path}`, { headers: stageHeaders, redirect: 'manual', signal: AbortSignal.timeout(60_000) });
    const html = await response.text();
    assert.equal(response.status, 200, path);
    assert.ok(!html.includes('NEXT_HTTP_ERROR_FALLBACK') && !html.includes('data-next-error'), `${path}: SSR failed`);
    assert.ok(!/"digest":"[0-9]+"/.test(html), `${path}: streamed SSR error`);
    if (path !== '/admin/settings') assert.ok(html.includes(credentials.username), `${path}: authenticated profile missing`);
  }
  const proxied = await fetch(`${origin.origin}/api/v1/me`, { headers: stageHeaders, signal: AbortSignal.timeout(10_000) });
  assert.equal(proxied.status, 200);
  assert.equal((await proxied.json()).user.username, credentials.username);
  const proxiedLogout = await fetch(`${origin.origin}/api/v1/auth/logout`, {
    method: 'POST', headers: { ...stageHeaders, Origin: origin.origin }, signal: AbortSignal.timeout(10_000),
  });
  assert.equal(proxiedLogout.status, 200);
  assert.match(proxiedLogout.headers.get('set-cookie') ?? '', /Max-Age=0/);
  console.log('PASS: stage proxy login/logout cookies, anonymous redirect, authenticated SSR dashboard/admin/settings');
}

const logout = await request('/auth/logout', { method: 'POST', headers });
assert.match(logout.headers.get('set-cookie') ?? '', /Max-Age=0/);
console.log('PASS: Go -> Minikube CRD health, signup/login/JWT cookie/me, dashboard/admin reads, logout');
console.log(`Local QA login saved to ${credentialsPath.pathname} (ignored by Git).`);
