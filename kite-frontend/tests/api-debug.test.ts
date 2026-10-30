import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { defaultDebugState, DEBUG_STATE_KEY } from '../src/api/debugMockData';
import { readDebugState } from '../src/api/debugMockState';

const values = new Map<string, string>();
const storage = {
  getItem: (key: string) => values.get(key) ?? null,
  setItem: (key: string, value: string) => { values.set(key, value); },
  removeItem: (key: string) => { values.delete(key); },
};
beforeEach(() => {
  values.clear();
  vi.stubGlobal('window', { localStorage: storage, location: { origin: 'https://kite.test' } });
  vi.stubEnv('NEXT_PUBLIC_KITE_MODE', 'debug'); vi.stubEnv('NEXT_PUBLIC_USE_MOCK', 'true');
  vi.resetModules();
});
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });
it.each([
  { ...defaultDebugState, users: [{ username: 'bad' }] },
  { ...defaultDebugState, vms: [{ id: 'bad' }] },
  { ...defaultDebugState, offers: [{ phase: 'unknown' }] },
  { ...defaultDebugState, config: { ...defaultDebugState.config, sshGateway: { status: { phase: 'Ready' } } } },
])('resets deeply corrupt persisted state %#', input => {
  // Given parseable JSON with corrupt nested data.
  storage.setItem(DEBUG_STATE_KEY, JSON.stringify(input));
  // When debug persistence crosses the storage boundary.
  const state = readDebugState();
  // Then the complete default replaces corrupt state.
  expect(state).toEqual(defaultDebugState);
  expect(JSON.parse(storage.getItem(DEBUG_STATE_KEY) ?? '{}')).toEqual(defaultDebugState);
});
it('runs a valid debug lifecycle through the same response contracts', async () => {
  // Given an explicit debug environment with isolated persistence.
  const { authApi, vmApi, adminApi, configApi } = await import('../src/api');
  // When the supported lifecycle executes through public adapters.
  const signup = await authApi.signup({ username: 'new', email: 'new@test', password: 'p', profile_image: '' });
  const login = await authApi.login({ email: 'admin', password: 'p' });
  const me = await authApi.getMe();
  const users = await adminApi.getUsers();
  const access = await adminApi.updateUserAccess('new', 1);
  const created = await vmApi.createVm({ name: 'life', domainPrefix: 'life', sshId: 'u', sshPassword: 'p', disk: 30 });
  const stopped = await vmApi.stopVm('life');
  const started = await vmApi.startVm('life');
  const updated = await vmApi.updateVm('life', { domainPrefix: 'updated' });
  const detail = await vmApi.getVm('life');
  const list = await vmApi.getVms();
  const ticket = await vmApi.createConsoleTicket('life');
  const forced = await adminApi.forceStopVm('hyeonseok-ns', 'life');
  const deleted = await adminApi.deleteVm('hyeonseok-ns', 'life');
  const personalDeleted = await vmApi.deleteVm('dev-vm-1');
  const offered = await adminApi.createVmOffer({ targetUser: 'new', cpu: 2, memory: '4Gi', disk: '30Gi' });
  const offers = await vmApi.getOffers();
  const claimed = await vmApi.claimOffer(offered.offer.name, { vmName: 'claimed', domainPrefix: 'claimed', sshId: 'u', initialLoginPassword: 'p' });
  await adminApi.deleteVmOffer('ns', 'unused');
  const domain = await adminApi.saveDomain('new.test');
  const https = await adminApi.saveHTTPSPolicy({ forceHttps: true });
  const contact = await adminApi.saveAdminContact('ops');
  const gateway = await adminApi.saveSSHGateway({ externalEnabled: true, externalPort: '2222', publicPort: '22' });
  const rotated = await adminApi.rotateRuntimeSecrets({ rotateJWTSecret: true, rotatePasswordSalt: true });
  const cert = await adminApi.saveCert({ tlsCert: 'cert', tlsKey: 'key' });
  const settings = await adminApi.getSettings();
  const config = await configApi.getConfig();
  const global = await adminApi.getVms();
  const removed = await adminApi.deleteUser('new');
  const logout = await authApi.logout();
  // Then every parsed result exposes the expected Go success semantics.
  expect(signup.user.access_level).toBe(0); expect(access.user.access_level).toBe(1);
  expect(login.user.access_level).toBe(3); expect(me.user.username).toBe('admin'); expect(users.users.length).toBeGreaterThan(0);
  expect(created.vm.name).toBe('life'); expect(stopped.vm.powerState).toBe('Off'); expect(started.vm.powerState).toBe('On');
  expect(updated.vm.domainPrefix).toBe('updated'); expect(detail.vm.name).toBe('life'); expect(list.vms.length).toBeGreaterThan(0); expect(ticket.ticket).toContain('life');
  expect(forced.vm.powerState).toBe('Off'); expect(deleted.vm.delete).toBe(true); expect(personalDeleted.message).toBeTruthy();
  expect(offers.offers.some(item => item.name === offered.offer.name)).toBe(true); expect(claimed.vm.name).toBe('claimed');
  expect(domain.config.baseDomain).toBe('new.test'); expect(https.config.forceHttps).toBe(true); expect(contact.config.adminContact).toBe('ops');
  expect(gateway.config.sshGateway.status.phase).toBe('Ready'); expect(rotated.config.hasJWTSecret).toBe(true); expect(cert.config.hasTLSCertificate).toBe(true);
  expect(settings.config.sshGateway.publicPort).toBe('22'); expect(config.config.sshGateway.status.phase).toBe('Ready');
  expect(global.vms.length).toBeGreaterThan(0); expect(removed.message).toBeTruthy(); expect(logout.message).toBeTruthy();
});
it('encodes websocket names and query values independently', async () => {
  // Given reserved URL characters and an HTTPS origin.
  vi.stubEnv('NEXT_PUBLIC_API_BASE_URL', '/api/v1');
  const { buildConsoleWebSocketUrl } = await import('../src/api/console');
  // When the connection URL is created.
  const url = new URL(buildConsoleWebSocketUrl('a/b ?', 't&?#'));
  // Then neither name nor ticket escapes its segment.
  expect(url.protocol).toBe('wss:'); expect(url.pathname).toBe('/api/v1/vms/a%2Fb%20%3F/console'); expect(url.searchParams.get('ticket')).toBe('t&?#');
});
