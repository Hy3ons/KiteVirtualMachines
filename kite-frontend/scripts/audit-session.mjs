import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { createServer } from 'node:http';

const users = {
  alpha: { username: 'review-alpha', email: 'alpha@example.test', namespace: 'alpha-ns', access_level: 1, profile_image: '' },
  beta: { username: 'review-beta', email: 'beta@example.test', namespace: 'beta-ns', access_level: 1, profile_image: '' },
  manager: { username: 'review-manager', email: 'manager@example.test', namespace: 'manager-ns', access_level: 2, profile_image: '' },
  viewer: { username: 'review-viewer', email: 'viewer@example.test', namespace: 'viewer-ns', access_level: 0, profile_image: '' },
  admin: { username: 'review-admin', email: 'admin@example.test', namespace: 'admin-ns', access_level: 3, profile_image: '' },
  malformed: { username: 'review-malformed', email: 'malformed@example.test', namespace: 'malformed-ns', access_level: 1, profile_image: '' },
};
const config = { baseDomain: 'example.test', forceHttps: true, adminContact: '', hasTLSCertificate: true, sshGateway: { externalEnabled: false, externalPort: '', publicPort: '', phase: 'Disabled', reason: 'ExternalDisabled', message: '' } };
const adminConfig = { ...config, hasJWTSecret: true, hasPasswordSalt: true, sshGateway: { externalEnabled: false, externalPort: '', publicPort: '', status: { phase: 'Disabled', reason: 'ExternalDisabled', message: '' } } };
const requests = [];

// vmFor supplies the complete Go-shaped VM contract for this isolated HTTP fixture.
// user identifies the owner; no request is forwarded to a live Go or Kubernetes service.
function vmFor(user) {
  return { id: `${user.namespace}/review-vm`, name: 'review-vm', namespace: user.namespace, owner: user.username, domain: '', phase: 'Running', powerState: 'On', currentPowerState: 'On', cpu: 2, memory: '4Gi', image: 'ubuntu-22.04', disk: '25Gi', domainPrefix: '', sshId: 'builder', delete: false, dataVolumePhase: 'Succeeded', dataVolumeProgress: '100%', dataVolumeMessage: '' };
}

const api = createServer((request, response) => {
  const account = request.headers.cookie?.split('=')[1];
  const user = Object.hasOwn(users, account) ? users[account] : null;
  requests.push({ method: request.method, path: request.url, account });
  response.setHeader('content-type', 'application/json');
  if (account === 'outage') { response.statusCode = 503; response.end('{"message":"fixture unavailable"}'); return; }
  if (!user) { response.statusCode = 401; response.end('{"message":"unauthorized"}'); return; }
  switch (request.url) {
    case '/api/v1/me': response.end(JSON.stringify({ user })); break;
    case '/api/v1/vms': response.end(JSON.stringify({ vms: account === 'malformed' ? [{ name: 'invalid-fixture' }] : [vmFor(user)] })); break;
    case '/api/v1/vms/review-vm': response.end(JSON.stringify({ vm: vmFor(user) })); break;
    case '/api/v1/vm-offers': response.end('{"offers":[]}'); break;
    case '/api/v1/config': response.end(JSON.stringify({ config })); break;
    case '/api/v1/admin/users': response.end(JSON.stringify({ users: Object.values(users) })); break;
    case '/api/v1/admin/vms': response.end(JSON.stringify({ vms: [vmFor(user)] })); break;
    case '/api/v1/admin/settings': response.end(JSON.stringify({ config: adminConfig })); break;
    default: response.statusCode = 404; response.end('{"message":"not found"}');
  }
});

// listenPort reserves an ephemeral local port and returns it after the server starts.
// server is a task-owned fixture or temporary port reservation, never an existing user service.
async function listenPort(server) {
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  const address = server.address();
  assert.ok(address && typeof address !== 'string');
  return address.port;
}

const apiPort = await listenPort(api);
const reservation = createServer();
const appPort = await listenPort(reservation);
await new Promise((resolve) => reservation.close(resolve));
const app = spawn(process.execPath, ['.next/standalone/server.js'], {
  env: { ...process.env, HOSTNAME: '127.0.0.1', PORT: String(appPort), KITE_API_INTERNAL_URL: `http://127.0.0.1:${apiPort}` },
  stdio: ['ignore', 'pipe', 'pipe'],
});

// request reads one production page without following redirects or dispatching browser events.
// account selects only a synthetic fixture cookie; no credentials are read from the environment.
function request(path, account) {
  return fetch(`http://127.0.0.1:${appPort}${path}`, { redirect: 'manual', headers: { 'user-agent': 'facebookexternalhit/1.1', ...(account ? { cookie: `accessToken=${account}` } : {}) }, signal: AbortSignal.timeout(15000) });
}

try {
  await new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('Production server startup timed out; run npm run build:prod first')), 15000);
    app.stdout.on('data', (data) => { if (data.toString().includes('Ready')) { clearTimeout(timer); resolve(); } });
    app.once('error', (error) => { clearTimeout(timer); reject(error); });
    app.once('exit', (code) => { clearTimeout(timer); reject(new Error(`Production server exited with ${code}`)); });
  });

  for (const path of ['/', '/kite-docs', '/privacy', '/terms', '/signup', '/login']) {
    const response = await request(path);
    assert.equal(response.status, 200, path);
    const html = await response.text();
    assert.equal((html.match(/<main\b/g) || []).length, 1, path);
    assert.ok(!/<button[^>]*class="[^"]*account-menu-trigger/.test(html), path);
    assert.ok(!/<nav[^>]*class="[^"]*demo-chapters/.test(html), path);
    if (path === '/') assert.ok(/<a[^>]*href="https:\/\/github.com\/hy3ons"[^>]*>Hy3ons<\/a>/.test(html));
  }
  assert.equal(requests.length, 0, 'Anonymous public pages must skip session and VM lookups');
  console.log('PASS 6 anonymous public pages: SSR, no account leak/debug chapters, footer link, no API reads');

  for (const path of ['/dashboard', '/admin/dashboard', '/admin/settings']) {
    const response = await request(path);
    assert.equal(response.status, 307, path);
    assert.equal(response.headers.get('location'), `/login?requireLogin=${encodeURIComponent(path)}`, path);
  }
  assert.equal(requests.length, 0);
  console.log('PASS anonymous private-page guards without backend requests');

  for (const path of ['/login', '/login?requireLogin=%2Fdashboard']) {
    const response = await request(path, 'alpha');
    assert.equal(response.status, 307, path);
    assert.equal(response.headers.get('location'), '/', path);
  }
  const expired = await request('/login', 'expired');
  assert.equal(expired.status, 200);
  console.log('PASS verified login -> landing; expired session -> login form');

  for (const path of ['/', '/kite-docs', '/privacy', '/terms']) {
    await Promise.all(['alpha', 'beta'].map(async (account) => {
      const response = await request(path, account);
      assert.equal(response.status, 200, path);
      assert.ok(response.headers.get('cache-control')?.includes('private'), path);
      const html = await response.text();
      assert.ok(/<button[^>]*class="[^"]*account-menu-trigger/.test(html), path);
      assert.ok(html.includes(users[account].email), path);
      assert.ok(!html.includes(users[account === 'alpha' ? 'beta' : 'alpha'].email), path);
    }));
  }
  assert.ok(requests.every((entry) => entry.path === '/api/v1/me'), 'Closed profile/demo must not fetch VMs');
  console.log('PASS concurrent public-session isolation, private caching and no closed-panel VM reads');

  const restricted = await request('/admin/settings', 'alpha');
  assert.equal(restricted.status, 307);
  assert.equal(restricted.headers.get('location'), '/dashboard');
  const dashboard = await request('/dashboard', 'alpha');
  assert.equal(dashboard.status, 200);
  assert.ok((await dashboard.text()).includes('review-vm'));
  const consoleStart = requests.length;
  const consolePage = await request('/dashboard/kite-machine/review-vm/console', 'alpha');
  assert.equal(consolePage.status, 200);
  assert.ok((await consolePage.text()).includes('<title>review-vm | 웹 콘솔 | Kite</title>'));
  assert.deepEqual(requests.slice(consoleStart).map(entry => entry.path).sort(), ['/api/v1/me', '/api/v1/vms/review-vm']);
  const manager = await request('/admin/dashboard', 'manager');
  assert.equal(manager.status, 200);
  assert.ok(requests.some((entry) => entry.path === '/api/v1/admin/users' && entry.account === 'manager'));
  assert.ok(!requests.some((entry) => entry.path === '/api/v1/admin/vms'));
  const detail = await request('/dashboard/kite-machine/review-vm', 'alpha');
  assert.equal(detail.status, 200);
  for (const account of ['viewer', 'manager']) {
    const settings = await request('/admin/settings', account);
    assert.equal(settings.status, 307);
    assert.equal(settings.headers.get('location'), '/dashboard');
  }
  assert.equal((await request('/dashboard', 'viewer')).status, 200);
  const viewerAdmin = await request('/admin/dashboard', 'viewer');
  assert.equal(viewerAdmin.status, 307);
  assert.equal(viewerAdmin.headers.get('location'), '/dashboard');
  for (const path of ['/admin/dashboard', '/admin/settings']) {
    const response = await request(path, 'admin');
    assert.equal(response.status, 200, path);
    assert.ok((await response.text()).includes(users.admin.email), path);
  }
  assert.ok(requests.some(entry => entry.path === '/api/v1/admin/vms' && entry.account === 'admin'));
  assert.ok(requests.some(entry => entry.path === '/api/v1/admin/settings' && entry.account === 'admin'));
  assert.equal((await request('/dashboard', 'malformed')).status, 500, 'Malformed successful data must fail SSR');
  const outage = await request('/dashboard', 'outage');
  assert.equal(outage.status, 500, 'A service outage must not become an anonymous redirect');
  assert.equal(outage.headers.get('location'), null);
  assert.ok(requests.every((entry) => entry.method === 'GET'), 'Review must never mutate resources');
  console.log('PASS dashboard/detail/console/admin SSR; role 0/1/2/3 guards; console skips settings; GET requests only');
  console.log('PASS malformed success and API outage fail SSR without pretending the user logged out');
} finally {
  const stopped = once(app, 'exit');
  if (app.exitCode === null && app.signalCode === null) { app.kill('SIGTERM'); await stopped; }
  await new Promise((resolve) => api.close(resolve));
}
