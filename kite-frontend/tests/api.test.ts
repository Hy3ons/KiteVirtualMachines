import { vm as fullVm, publicConfig, adminConfig } from './fixtures/api';
import { createServer } from 'node:http';
import type { IncomingMessage, ServerResponse } from 'node:http';
import { once } from 'node:events';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError, createApiClient } from '../src/api/client';
import { readSession } from '../src/api/session';
import { loadAdminDashboard, loadDashboard } from '../src/api/pageData';
import { parseAccountVmUsage } from '../src/api/accountUsage';

let baseUrl = '';
const requests: { readonly method: string; readonly path: string; readonly cookie: string; readonly body: string }[] = [];
let reply: (request: IncomingMessage, response: ServerResponse) => void;
const server = createServer(async (request, response) => {
  request.setEncoding('utf8');
  let body = '';
  for await (const chunk of request) body += String(chunk);
  requests.push({ method: request.method || '', path: request.url || '', cookie: request.headers.cookie || '', body });
  response.setHeader('Content-Type', 'application/json');
  reply(request, response);
});

beforeAll(async () => {
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  const address = server.address();
  if (!address || typeof address === 'string') throw new TypeError('Expected TCP test server');
  baseUrl = `http://127.0.0.1:${address.port}/api/v1`;
});

afterAll(() => new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve())));
beforeEach(() => {
  requests.length = 0;
  reply = (_request, response) => response.end(JSON.stringify({ message: 'ok' }));
});
afterEach(() => vi.unstubAllEnvs());

describe('fetch HTTP transport', () => {
  it.each(['post', 'patch', 'delete'] as const)('sends %s with the correct method and JSON body', async (method) => {
    // Given a real HTTP endpoint and a cookie scoped to this SSR request.
    const client = createApiClient(baseUrl, { cookie: 'accessToken="Bearer fixture"' });
    // When a mutation is sent through the same transport used by the UI.
    if (method === 'delete') await client.delete('/vms/demo');
    else await client[method]('/vms/demo', { powerState: 'Off' });
    // Then the server receives the expected wire contract.
    expect(requests).toEqual([{ method: method.toUpperCase(), path: '/api/v1/vms/demo', cookie: 'accessToken="Bearer fixture"', body: method === 'delete' ? '' : '{"powerState":"Off"}' }]);
  });

  it('isolates cookies for concurrent SSR users', async () => {
    // Given two separate request-scoped clients.
    const first = createApiClient(baseUrl, { cookie: 'accessToken=first' });
    const second = createApiClient(baseUrl, { cookie: 'accessToken=second' });
    // When both fetch private data concurrently.
    await Promise.all([first.get('/vms'), second.get('/vms')]);
    // Then neither client overwrites the other cookie.
    expect(requests.map((request) => request.cookie).sort()).toEqual(['accessToken=first', 'accessToken=second']);
  });

  it.each([401, 403, 500])('preserves backend error %s without retrying mutations', async (status) => {
    // Given a rejected backend mutation.
    reply = (_request, response) => { response.statusCode = status; response.end('{"message":"backend rejected"}'); };
    const unauthorized = vi.fn();
    const client = createApiClient(baseUrl, { onUnauthorized: unauthorized });
    // When the mutation fails.
    const result = client.post('/vms', { name: 'demo' });
    // Then the caller sees the original status/message; only 401 expires the session.
    await expect(result).rejects.toMatchObject({ status, message: 'backend rejected' });
    expect(requests).toHaveLength(1);
    expect(unauthorized).toHaveBeenCalledTimes(status === 401 ? 1 : 0);
  });

  it('keeps invalid login credentials on the login screen', async () => {
    // Given an authentication failure rather than an expired application session.
    reply = (_request, response) => { response.statusCode = 401; response.end('{"message":"invalid email or password"}'); };
    const unauthorized = vi.fn();
    // When login is attempted.
    const result = createApiClient(baseUrl, { onUnauthorized: unauthorized }).post('/auth/login', { email: 'a', password: 'b' });
    // Then the form can show the error without triggering a redirect loop.
    await expect(result).rejects.toBeInstanceOf(ApiError);
    expect(unauthorized).not.toHaveBeenCalled();
  });

  it('aborts requests that exceed the timeout', async () => {
    // Given an endpoint that never responds.
    reply = () => undefined;
    // When its request budget expires.
    const result = createApiClient(baseUrl, { timeout: 30 }).get('/vms');
    // Then the request fails instead of hanging the SSR render.
    await expect(result).rejects.toMatchObject({ name: 'TimeoutError' });
  });
});

describe('Go-backed session validation', () => {
  it('skips /me when the request has no JWT cookie', async () => {
    // Given an anonymous request.
    const client = createApiClient(baseUrl);
    // When the server checks the session.
    const user = await readSession(client, false);
    // Then no network lookup or synthetic login occurs.
    expect(user).toBeNull();
    expect(requests).toHaveLength(0);
  });

  it('uses the current backend role and strips non-public session fields', async () => {
    // Given a stale admin JWT whose current backend role is read-only.
    reply = (_request, response) => response.end(JSON.stringify({ user: { username: 'viewer', email: 'viewer@example.test', namespace: 'viewer-ns', access_level: 0, password: 'never serialize', accessToken: 'never serialize' } }));
    // When /me verifies that cookie.
    const user = await readSession(createApiClient(baseUrl, { cookie: 'accessToken=stale-admin' }), true);
    // Then only the current safe profile is passed to React.
    expect(user).toEqual({ username: 'viewer', email: 'viewer@example.test', namespace: 'viewer-ns', access_level: 0, profile_image: '' });
    expect(requests[0]?.cookie).toBe('accessToken=stale-admin');
  });

  it.each([401, 403, 404])('returns no session when Go rejects it with %s', async (status) => {
    // Given an expired, revoked or deleted backend account.
    reply = (_request, response) => { response.statusCode = status; response.end('{"message":"invalid session"}'); };
    // When the server verifies the session.
    const user = await readSession(createApiClient(baseUrl), true);
    // Then access cannot be authorized from browser storage.
    expect(user).toBeNull();
  });

  it('propagates backend outages instead of pretending the user logged out', async () => {
    // Given an unavailable API service.
    reply = (_request, response) => { response.statusCode = 503; response.end('{"message":"unavailable"}'); };
    // When /me is requested.
    const result = readSession(createApiClient(baseUrl), true);
    // Then the SSR error boundary can offer retry.
    await expect(result).rejects.toMatchObject({ status: 503 });
  });

  it('rejects malformed identity data', async () => {
    // Given a response with no namespace and an invalid permission value.
    reply = (_request, response) => response.end('{"user":{"username":"admin","access_level":99}}');
    // When /me crosses the trust boundary.
    const result = readSession(createApiClient(baseUrl), true);
    // Then no invalid session is rendered.
    await expect(result).rejects.toMatchObject({ name: 'ZodError' });
  });
});

describe('SSR reads using Go response contracts', () => {
  it('loads VM lists, offers and settings from their existing endpoints', async () => {
    // Given response fields used by the Go API, including controller failure states.
    const vm = { id: 'ns/demo', name: 'demo', namespace: 'ns', owner: 'ns', domain: '', phase: 'Failed', powerState: 'On', currentPowerState: '', cpu: 2, memory: '4Gi', image: 'ubuntu', disk: '30Gi', domainPrefix: '', sshId: 'ubuntu', delete: false, dataVolumePhase: 'Failed', dataVolumeProgress: '', dataVolumeMessage: 'import failed' };
    reply = (request, response) => response.end(JSON.stringify(request.url?.endsWith('/vms') ? { vms: [vm] } : request.url?.endsWith('/vm-offers') ? { offers: [] } : { config: publicConfig }));
    // When the dashboard loads on the server.
    const data = await loadDashboard(createApiClient(baseUrl));
    // Then data reaches the SSR screen without losing VM failure details.
    expect(data.vms[0]).toEqual(vm);
    expect(data.config.baseDomain).toBe('example.test');
    expect(requests.map((request) => request.path).sort()).toEqual(['/api/v1/config', '/api/v1/vm-offers', '/api/v1/vms']);
  });

  it('supports Go users without status and avoids admin VM reads for managers', async () => {
    // Given a level-2 manager and Go PublicUser fields (no UI-only status field).
    reply = (_request, response) => response.end('{"users":[{"username":"viewer","email":"v@example.test","namespace":"v-ns","access_level":0}]}');
    // When the manager dashboard loads.
    const data = await loadAdminDashboard(createApiClient(baseUrl), 2);
    // Then the adapter supplies UI fields without calling a level-3 endpoint.
    expect(data.users[0]).toMatchObject({ accessLevel: 0, status: 'Pending' });
    expect(data.vms).toEqual([]);
    expect(requests.map((request) => request.path)).toEqual(['/api/v1/admin/users']);
  });
});

describe('browser API adapters', () => {
  it.each([
    { vms: [{ ...fullVm, phase: 'Running', delete: false }, { ...fullVm, phase: 'Stopped', delete: false }, { ...fullVm, phase: 'Running', delete: true }], usage: { total: 2, running: 1 } },
    { vms: [], usage: { total: 0, running: 0 } },
  ])('loads account VM counts from the existing personal VM endpoint: $usage', async ({ vms, usage }) => {
    vi.stubEnv('NEXT_PUBLIC_API_BASE_URL', baseUrl);
    vi.stubEnv('NEXT_PUBLIC_USE_MOCK', 'false');
    vi.resetModules();
    reply = (_request, response) => response.end(JSON.stringify({ vms }));
    const { vmApi } = await import('../src/api');
    expect(parseAccountVmUsage(await vmApi.getVms())).toEqual(usage);
    expect(requests).toHaveLength(1);
    expect(requests[0]).toMatchObject({ method: 'GET', path: '/api/v1/vms' });
  });

  it('rejects a malformed VM-count response instead of displaying a false zero', async () => {
    vi.stubEnv('NEXT_PUBLIC_API_BASE_URL', baseUrl);
    vi.stubEnv('NEXT_PUBLIC_USE_MOCK', 'false');
    vi.resetModules();
    reply = (_request, response) => response.end('{"vms":null}');
    const { vmApi } = await import('../src/api');
    await expect(vmApi.getVms()).rejects.toMatchObject({ name: 'ZodError' });
  });

  it.each([true, false])('saves gateway exposure with externalEnabled=%s and separate Service/public ports', async (externalEnabled) => {
    // Given the gateway editor's payload and a Go-shaped settings response.
    vi.stubEnv('NEXT_PUBLIC_API_BASE_URL', baseUrl);
    vi.stubEnv('NEXT_PUBLIC_USE_MOCK', 'false');
    vi.resetModules();
    const payload = { externalEnabled, externalPort: '12311', publicPort: '22' };
    reply = (_request, response) => response.end(JSON.stringify({ message: 'updated', config: { ...adminConfig, sshGateway: { ...payload, status: adminConfig.sshGateway.status } } }));
    const { adminApi } = await import('../src/api');
    // When activation or deactivation is saved from the editor.
    const result = await adminApi.saveSSHGateway(payload);
    // Then Go receives both distinct ports and the desired enabled state unchanged.
    expect(requests[0]).toMatchObject({ method: 'POST', path: '/api/v1/admin/ssh-gateway' });
    expect(JSON.parse(requests[0]?.body || '{}')).toEqual(payload);
    expect(result.config.sshGateway).toMatchObject(payload);
  });

  it.each([
    ['getVms', 'GET', '/vms'], ['startVm', 'POST', '/vms/demo/start'],
    ['stopVm', 'POST', '/vms/demo/stop'], ['deleteVm', 'DELETE', '/vms/demo'],
    ['createConsoleTicket', 'POST', '/vms/demo/console-ticket'],
  ] as const)('maps %s directly to the Go endpoint', async (method, expectedMethod, endpoint) => {
    // Given the real UI adapters pointed at our HTTP fixture rather than a browser mock.
    vi.stubEnv('NEXT_PUBLIC_API_BASE_URL', baseUrl);
    vi.stubEnv('NEXT_PUBLIC_USE_MOCK', 'false');
    vi.resetModules();
    reply = (_request, response) => response.end(JSON.stringify(method === 'getVms' ? { vms: [fullVm] } : method === 'createConsoleTicket' ? { ticket: 'fixture', expiresAt: '2030-01-01T00:00:00Z' } : { message: 'ok', vm: fullVm }));
    const { vmApi } = await import('../src/api');
    // When a UI operation calls its adapter.
    await vmApi[method]('demo');
    // Then the request reaches the Go route with the expected HTTP method.
    expect(requests[0]).toMatchObject({ method: expectedMethod, path: '/api/v1' + endpoint });
  });

  it('sends the existing login payload and exposes only the safe profile', async () => {
    // Given the real auth adapter and Go's cookie-based response shape.
    vi.stubEnv('NEXT_PUBLIC_API_BASE_URL', baseUrl);
    vi.stubEnv('NEXT_PUBLIC_USE_MOCK', 'false');
    vi.resetModules();
    reply = (_request, response) => {
      response.setHeader('Set-Cookie', 'accessToken="Bearer fixture"; Path=/; HttpOnly; SameSite=Lax');
      response.end('{"expiresIn":600,"expiresAt":"2026-10-30T00:00:00Z","user":{"username":"demo","email":"demo@example.test","namespace":"demo-ns","access_level":1,"accessToken":"must not hydrate"}}');
    };
    const { authApi } = await import('../src/api');
    // When login is submitted.
    const result = await authApi.login({ email: 'demo@example.test', password: 'fixture-password' });
    // Then field names match Go and only validated profile fields reach the app.
    expect(JSON.parse(requests[0]?.body || '{}')).toEqual({ email: 'demo@example.test', password: 'fixture-password' });
    expect(result.user).toEqual({ username: 'demo', email: 'demo@example.test', namespace: 'demo-ns', access_level: 1, profile_image: '' });
  });
});
