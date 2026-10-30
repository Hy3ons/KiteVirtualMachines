import { execFileSync } from 'node:child_process';
import { randomBytes } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { beforeAll, describe, expect, test } from 'vitest';
import { z } from 'zod';
import { createApiClient } from '../src/api/client';
import { createEndpoints } from '../src/api/endpoints';
import { authResponseSchema } from '../src/api/contracts';

const api = 'http://127.0.0.1:8080/api/v1';
const frontend = 'http://127.0.0.1:5173';
const kubectl = '/opt/homebrew/bin/kubectl';
const profile = 'kite-local-qa';
const privateDirectory = new URL('../../.local-dev/kite-local-qa/', import.meta.url);
const credentialsSchema = z.object({ username: z.string(), email: z.email(), password: z.string().min(16), profile_image: z.string() });
type Credentials = z.infer<typeof credentialsSchema>;
type Endpoints = ReturnType<typeof createEndpoints>;
const anonymous = createEndpoints(createApiClient(api));
let admin: Endpoints;
let member: Endpoints;
let adminCredentials: Credentials;
let memberCredentials: Credentials;
let adminCookie = '';
let memberCookie = '';

// loadAccount creates only named QA accounts, keeping passwords in ignored owner-only files.
// Existing credentials allow reruns without resetting the first-administrator database.
async function loadAccount(kind: 'admin' | 'member', expectedLevel: 0 | 3): Promise<Credentials> {
  const path = new URL(`${kind}.json`, privateDirectory);
  try {
    return credentialsSchema.parse(JSON.parse(await readFile(path, 'utf8')));
  } catch (error) {
    if (!(error instanceof Error) || !('code' in error) || error.code !== 'ENOENT') throw error;
  }
  const credentials = { username: `qa-${kind}`, email: `qa-${kind}@kite.local`, password: randomBytes(24).toString('base64url'), profile_image: '' };
  const signup = await anonymous.authApi.signup(credentials);
  await mkdir(privateDirectory, { recursive: true, mode: 0o700 });
  await writeFile(path, `${JSON.stringify(credentials, null, 2)}\n`, { mode: 0o600 });
  expect(signup.user.access_level).toBe(expectedLevel);
  return credentials;
}

// login checks the real Go cookie contract and returns a cookie scoped to this test process.
async function login(credentials: Credentials, baseUrl = api): Promise<string> {
  const response = await fetch(`${baseUrl}/auth/login`, {
    method: 'POST', headers: { 'Content-Type': 'application/json', Origin: frontend },
    body: JSON.stringify(credentials), signal: AbortSignal.timeout(10_000),
  });
  expect(response.status).toBe(200);
  expect(authResponseSchema.parse(await response.json()).user.email).toBe(credentials.email);
  const header = response.headers.get('set-cookie') ?? '';
  expect(header.includes('HttpOnly')).toBe(true);
  expect(header.includes('SameSite=Lax')).toBe(true);
  expect(header.includes('Secure')).toBe(false);
  const cookie = header.split(';')[0];
  if (!cookie?.startsWith('accessToken=')) throw new Error('Missing local session cookie');
  return cookie;
}

// page requests SSR HTML without a browser, checking redirects and streamed render failures.
async function page(path: string, cookie = ''): Promise<Response> {
  return fetch(`${frontend}${path}`, { headers: { Cookie: cookie }, redirect: 'manual', signal: AbortSignal.timeout(60_000) });
}

beforeAll(async () => {
  expect(execFileSync(kubectl, ['config', 'current-context'], { encoding: 'utf8', timeout: 10_000 }).trim()).toBe(profile);
  adminCredentials = await loadAccount('admin', 3);
  memberCredentials = await loadAccount('member', 0);
  adminCookie = await login(adminCredentials);
  memberCookie = await login(memberCredentials);
  admin = createEndpoints(createApiClient(api, { cookie: adminCookie }));
  member = createEndpoints(createApiClient(api, { cookie: memberCookie }));
});

describe('real Next -> Go -> isolated Minikube', () => {
  test('first signup is Level 3; subsequent signup is Level 0; secrets stay private', async () => {
    expect((await admin.authApi.getMe()).user.access_level).toBe(3);
    expect((await member.authApi.getMe()).user.access_level).toBe(0);
    const response = await fetch(`${api}/me`, { headers: { Cookie: adminCookie } });
    const body = await response.text();
    expect(body).not.toMatch(/password|passwordHash|jwtSecret/i);
    await expect(anonymous.authApi.getMe()).rejects.toMatchObject({ status: 401 });
    await expect(anonymous.authApi.login({ email: adminCredentials.email, password: 'wrong-password' })).rejects.toMatchObject({ status: 401 });
  });

  test('dashboard and settings responses pass the production Zod adapters', async () => {
    await admin.configApi.getConfig();
    await admin.vmApi.getVms();
    await admin.vmApi.getOffers();
    expect((await admin.adminApi.getUsers()).users.length).toBeGreaterThanOrEqual(2);
    await admin.adminApi.getVms();
    await admin.adminApi.getSettings();
    await expect(member.adminApi.getSettings()).rejects.toMatchObject({ status: 403 });
    await expect(member.vmApi.createVm({ name: 'forbidden', domainPrefix: 'forbidden', sshId: 'qa', sshPassword: randomBytes(24).toString('hex'), disk: 10 })).rejects.toMatchObject({ status: 403 });
  });

  test('real proxy cookies, authenticated SSR, route guards and logout', async () => {
    const cookie = await login(adminCredentials, `${frontend}/api/v1`);
    expect((await page('/dashboard')).status).toBe(307);
    const loggedInLogin = await page('/login', cookie);
    expect(loggedInLogin.status).toBe(307);
    expect(loggedInLogin.headers.get('location')).toBe('/');
    for (const path of ['/dashboard', '/admin/dashboard', '/admin/settings']) {
      const response = await page(path, cookie);
      const html = await response.text();
      expect(response.status, path).toBe(200);
      expect(html, path).not.toMatch(/NEXT_HTTP_ERROR_FALLBACK|data-next-error|"digest":"[0-9]+"/);
      expect(html, path).toContain(adminCredentials.username);
    }
    expect((await page('/admin/settings', memberCookie)).status).toBe(307);
    const proxied = createEndpoints(createApiClient(`${frontend}/api/v1`, { cookie }));
    expect((await proxied.authApi.getMe()).user.email).toBe(adminCredentials.email);
    const logout = await fetch(`${frontend}/api/v1/auth/logout`, { method: 'POST', headers: { Cookie: cookie, Origin: frontend } });
    expect(logout.status).toBe(200);
    expect((logout.headers.get('set-cookie') ?? '').includes('Max-Age=0')).toBe(true);
  });

  test('permission changes include Level 0 and take effect without re-login', async () => {
    try {
      expect((await admin.adminApi.updateUserAccess(memberCredentials.username, 1)).user.access_level).toBe(1);
      expect((await member.authApi.getMe()).user.access_level).toBe(1);
      expect((await admin.adminApi.updateUserAccess(memberCredentials.username, 0)).user.access_level).toBe(0);
      expect((await member.authApi.getMe()).user.access_level).toBe(0);
    } finally {
      await admin.adminApi.updateUserAccess(memberCredentials.username, 0);
    }
  });

  test('VM and assigned-offer CRUD persist desired state in real CRDs', async () => {
    for (const endpoint of [admin, member]) {
      const { user } = await endpoint.authApi.getMe();
      execFileSync(kubectl, ['--context', profile, 'apply', '-f', '-'], { input: JSON.stringify({ apiVersion: 'v1', kind: 'Namespace', metadata: { name: user.namespace } }), stdio: ['pipe', 'pipe', 'pipe'], timeout: 10_000 });
    }
    const suffix = randomBytes(4).toString('hex');
    const name = `qa-vm-${suffix}`;
    const { vm } = await admin.vmApi.createVm({ name, domainPrefix: name, sshId: `qa${suffix}`, sshPassword: randomBytes(24).toString('hex'), cpu: 2, memory: '4Gi', disk: '20Gi', powerState: 'Off' });
    try {
      expect(vm.name).toBe(name);
      expect((await admin.vmApi.getVm(name)).vm.disk).toBe('20Gi');
      expect((await admin.vmApi.startVm(name)).vm.powerState).toBe('On');
      expect((await admin.vmApi.stopVm(name)).vm.powerState).toBe('Off');
      await expect(admin.vmApi.createConsoleTicket(name)).rejects.toMatchObject({ status: 409 });
      expect((await admin.vmApi.deleteVm(name)).vm.delete).toBe(true);
    } finally {
      execFileSync(kubectl, ['--context', profile, '-n', vm.namespace, 'delete', 'kitevirtualmachine', name, '--ignore-not-found', '--wait=false'], { stdio: 'pipe', timeout: 10_000 });
    }
    const { offer } = await admin.adminApi.createVmOffer({ targetUser: memberCredentials.username, cpu: 2, memory: '4Gi', disk: '20Gi' });
    const claimedName = `qa-offer-${suffix}`;
    try {
      expect((await member.vmApi.getOffers()).offers.some(item => item.name === offer.name)).toBe(true);
      const { vm: claimed } = await member.vmApi.claimOffer(offer.name, { vmName: claimedName, domainPrefix: claimedName, sshId: `offer${suffix}`, initialLoginPassword: randomBytes(24).toString('hex'), powerState: 'Off' });
      expect(claimed.cpu).toBe(2);
      expect(claimed.disk).toBe('20Gi');
      expect((await member.vmApi.getOffers()).offers.some(item => item.name === offer.name)).toBe(false);
    } finally {
      // Successful claim consumes the offer, so cleanup also tolerates its absence.
      execFileSync(kubectl, ['--context', profile, '-n', offer.namespace, 'delete', 'kitevirtualmachine', claimedName, '--ignore-not-found', '--wait=false'], { stdio: 'pipe', timeout: 10_000 });
      execFileSync(kubectl, ['--context', profile, '-n', offer.namespace, 'delete', 'kitevirtualmachineoffer', offer.name, '--ignore-not-found', '--wait=false'], { stdio: 'pipe', timeout: 10_000 });
    }
    const { offer: unused } = await admin.adminApi.createVmOffer({ targetUser: memberCredentials.username, cpu: 2, memory: '4Gi', disk: '20Gi' });
    await admin.adminApi.deleteVmOffer(unused.namespace, unused.name);
  });
});
