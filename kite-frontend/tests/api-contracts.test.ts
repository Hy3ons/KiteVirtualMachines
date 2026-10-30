import { createServer } from 'node:http';
import { once } from 'node:events';
import { afterAll, beforeAll, beforeEach, expect, it } from 'vitest';
import { createApiClient } from '../src/api/client';
import { createEndpoints } from '../src/api/endpoints';
import { user, vm, offer, publicConfig, adminConfig } from './fixtures/api';

type Endpoints = ReturnType<typeof createEndpoints>;
const create = { name: 'demo', domainPrefix: 'demo', sshId: 'ubuntu', sshPassword: 'secret', disk: 30 };
const cases: readonly { readonly name: string; readonly method: string; readonly path: string; readonly response: unknown; readonly run: (api: Endpoints) => Promise<unknown> }[] = [
  { name: 'login', method: 'POST', path: '/auth/login', response: { user, expiresIn: 3600, expiresAt: '2030-01-01T00:00:00Z' }, run: a => a.authApi.login({ email: 'demo', password: 'secret' }) },
  { name: 'signup', method: 'POST', path: '/auth/signup', response: { user, message: 'created' }, run: a => a.authApi.signup({ username: 'demo', email: 'demo', password: 'secret', profile_image: '' }) },
  { name: 'logout', method: 'POST', path: '/auth/logout', response: { message: 'out' }, run: a => a.authApi.logout() },
  { name: 'me', method: 'GET', path: '/me', response: { user, expiresAt: '2030-01-01T00:00:00Z' }, run: a => a.authApi.getMe() },
  { name: 'public config', method: 'GET', path: '/config', response: { config: publicConfig }, run: a => a.configApi.getConfig() },
  { name: 'vms', method: 'GET', path: '/vms', response: { vms: [vm] }, run: a => a.vmApi.getVms() },
  { name: 'vm', method: 'GET', path: '/vms/demo', response: { vm }, run: a => a.vmApi.getVm('demo') },
  { name: 'create', method: 'POST', path: '/vms', response: { vm }, run: a => a.vmApi.createVm(create) },
  { name: 'update', method: 'PATCH', path: '/vms/demo', response: { vm }, run: a => a.vmApi.updateVm('demo', { powerState: 'Off' }) },
  { name: 'start', method: 'POST', path: '/vms/demo/start', response: { vm }, run: a => a.vmApi.startVm('demo') },
  { name: 'stop', method: 'POST', path: '/vms/demo/stop', response: { vm }, run: a => a.vmApi.stopVm('demo') },
  { name: 'delete', method: 'DELETE', path: '/vms/demo', response: { message: 'deleted', vm }, run: a => a.vmApi.deleteVm('demo') },
  { name: 'ticket', method: 'POST', path: '/vms/demo/console-ticket', response: { ticket: 't', expiresAt: '2030' }, run: a => a.vmApi.createConsoleTicket('demo') },
  { name: 'offers', method: 'GET', path: '/vm-offers', response: { offers: [offer] }, run: a => a.vmApi.getOffers() },
  { name: 'claim', method: 'POST', path: '/vm-offers/demo/claim', response: { vm }, run: a => a.vmApi.claimOffer('demo', { vmName: 'demo', domainPrefix: 'demo', sshId: 'u', initialLoginPassword: 'p' }) },
  { name: 'users', method: 'GET', path: '/admin/users', response: { users: [user] }, run: a => a.adminApi.getUsers() },
  { name: 'access', method: 'PATCH', path: '/admin/users/demo/access-level', response: { user }, run: a => a.adminApi.updateUserAccess('demo', 2) },
  { name: 'delete user', method: 'DELETE', path: '/admin/users/demo', response: { message: 'deleted' }, run: a => a.adminApi.deleteUser('demo') },
  { name: 'global vms', method: 'GET', path: '/admin/vms', response: { vms: [vm] }, run: a => a.adminApi.getVms() },
  { name: 'force stop', method: 'PATCH', path: '/admin/vms/ns/demo/power', response: { vm }, run: a => a.adminApi.forceStopVm('ns', 'demo') },
  { name: 'admin delete vm', method: 'DELETE', path: '/admin/vms/ns/demo', response: { message: 'deleted', vm }, run: a => a.adminApi.deleteVm('ns', 'demo') },
  { name: 'settings', method: 'GET', path: '/admin/settings', response: { config: adminConfig }, run: a => a.adminApi.getSettings() },
  { name: 'domain', method: 'POST', path: '/admin/domain', response: { message: 'ok', config: adminConfig }, run: a => a.adminApi.saveDomain('example.test') },
  { name: 'https', method: 'POST', path: '/admin/https', response: { message: 'ok', config: adminConfig }, run: a => a.adminApi.saveHTTPSPolicy({ forceHttps: true }) },
  { name: 'gateway', method: 'POST', path: '/admin/ssh-gateway', response: { message: 'ok', config: adminConfig }, run: a => a.adminApi.saveSSHGateway({ externalEnabled: false, externalPort: '', publicPort: '' }) },
  { name: 'contact', method: 'POST', path: '/admin/admin-contact', response: { message: 'ok', config: adminConfig }, run: a => a.adminApi.saveAdminContact('ops') },
  { name: 'create offer', method: 'POST', path: '/admin/vm-offers', response: { offer }, run: a => a.adminApi.createVmOffer({ cpu: 2, memory: '4Gi', disk: '30Gi' }) },
  { name: 'delete offer', method: 'DELETE', path: '/admin/vm-offers/ns/demo', response: { message: 'deleted' }, run: a => a.adminApi.deleteVmOffer('ns', 'demo') },
  { name: 'rotate', method: 'POST', path: '/admin/runtime-secrets/rotate', response: { message: 'ok', config: adminConfig }, run: a => a.adminApi.rotateRuntimeSecrets({ rotateJWTSecret: true }) },
  { name: 'cert', method: 'POST', path: '/admin/cert', response: { message: 'ok', config: adminConfig }, run: a => a.adminApi.saveCert({ tlsCert: 'cert', tlsKey: 'key' }) },
];
let responseBody = '{}';
let status = 200;
let hang = false;
let baseUrl = '';
const requests: { method: string; path: string }[] = [];
const server = createServer((request, response) => {
  requests.push({ method: request.method ?? '', path: request.url ?? '' });
  if (hang) return;
  response.statusCode = status;
  response.end(responseBody);
});
beforeAll(async () => {
  server.listen(0, '127.0.0.1'); await once(server, 'listening');
  const address = server.address();
  if (!address || typeof address === 'string') throw new TypeError('Expected TCP listener');
  baseUrl = `http://127.0.0.1:${address.port}`;
});
afterAll(() => new Promise<void>((resolve, reject) => { server.closeAllConnections(); server.close(error => error ? reject(error) : resolve()); }));
beforeEach(() => { requests.length = 0; status = 200; hang = false; });
it.each(cases)('accepts Go success for $name', async scenario => {
  // Given the actual endpoint response shape.
  responseBody = JSON.stringify(scenario.response);
  // When the shared browser/SSR operation executes.
  const result = await scenario.run(createEndpoints(createApiClient(baseUrl)));
  // Then a validated result and exactly one expected request are observable.
  expect(result).toBeDefined();
  expect(requests).toEqual([{ method: scenario.method, path: scenario.path }]);
});
it.each(cases)('rejects malformed 2xx for $name', async scenario => {
  // Given a valid JSON body missing the endpoint's required fields.
  responseBody = '{}';
  // When the shared operation crosses its response boundary.
  const result = scenario.run(createEndpoints(createApiClient(baseUrl)));
  // Then malformed data cannot reach consumers.
  await expect(result).rejects.toMatchObject({ name: 'ZodError' });
  expect(requests).toHaveLength(1);
});
it.each([
  { config: { ...publicConfig, hasTLSCertificate: undefined } },
  { config: { ...publicConfig, sshGateway: { ...publicConfig.sshGateway, phase: undefined } } },
  { config: { ...publicConfig, sshGateway: { ...publicConfig.sshGateway, externalEnabled: 'false' } } },
])('rejects malformed public settings %#', async body => {
  // Given a missing required field or wrong nested type.
  responseBody = JSON.stringify(body);
  // When public settings are loaded.
  const result = createEndpoints(createApiClient(baseUrl)).configApi.getConfig();
  // Then the invalid success is rejected.
  await expect(result).rejects.toMatchObject({ name: 'ZodError' });
});
it.each(['hasJWTSecret', 'hasPasswordSalt', 'hasTLSCertificate'])('requires admin flag %s', async key => {
  // Given an admin response missing one flag.
  responseBody = JSON.stringify({ config: { ...adminConfig, [key]: undefined } });
  // When admin settings are loaded.
  const result = createEndpoints(createApiClient(baseUrl)).adminApi.getSettings();
  // Then the missing flag is rejected.
  await expect(result).rejects.toMatchObject({ name: 'ZodError' });
});
it.each([-1, 4, 1.5, '1'])('rejects invalid role %s across identity and user lists', async role => {
  // Given an invalid wire role.
  responseBody = JSON.stringify({ user: { ...user, access_level: role }, users: [{ ...user, access_level: role }] });
  // When both identity boundaries parse it.
  const api = createEndpoints(createApiClient(baseUrl));
  const results = await Promise.allSettled([api.authApi.getMe(), api.adminApi.getUsers()]);
  // Then neither accepts a privilege outside the shared enum.
  expect(results.map(result => result.status)).toEqual(['rejected', 'rejected']);
});
it.each([-1, 0.5, '600'])('rejects invalid session lifetime %s', async expiresIn => {
  // Given a login success containing an invalid integer lifetime.
  responseBody = JSON.stringify({ user, expiresIn, expiresAt: '2030-01-01T00:00:00Z' });
  // When authentication parses the server response.
  const result = createEndpoints(createApiClient(baseUrl)).authApi.login({ email: 'demo', password: 'secret' });
  // Then invalid expiration data cannot become an authenticated profile.
  await expect(result).rejects.toMatchObject({ name: 'ZodError' });
});
it('preserves ticket cancellation and encodes the VM segment', async () => {
  // Given a hanging console endpoint and an abortable request.
  hang = true;
  const controller = new AbortController();
  const requestReceived = once(server, 'request');
  // When the ticket is cancelled after reaching the server.
  const result = createEndpoints(createApiClient(baseUrl)).vmApi.createConsoleTicket('a/b ?#', { signal: controller.signal });
  await requestReceived; controller.abort();
  // Then cancellation reaches the transport and names stay one encoded segment.
  await expect(result).rejects.toMatchObject({ name: 'AbortError' });
  expect(requests[0]?.path).toBe('/vms/a%2Fb%20%3F%23/console-ticket');
});
it.each([401, 403, 500])('falls back for non-JSON error %s without retries', async errorStatus => {
  // Given a proxy-generated HTML failure.
  status = errorStatus; responseBody = '<html>Unavailable</html>';
  // When an operation runs.
  const result = createEndpoints(createApiClient(baseUrl)).vmApi.startVm('demo');
  // Then status survives and no retry duplicates the operation.
  await expect(result).rejects.toMatchObject({ status: errorStatus, message: `API request failed (${errorStatus})` });
  expect(requests).toHaveLength(1);
});
it('reports non-JSON success with safe operation context', async () => {
  // Given an HTML success from a misrouted proxy.
  responseBody = '<html>private data</html>';
  // When a VM operation expects JSON.
  const result = createEndpoints(createApiClient(baseUrl)).vmApi.getVms();
  // Then the error identifies the operation without exposing response content.
  await expect(result).rejects.toMatchObject({ name: 'ApiContractError', method: 'GET', endpoint: '/vms', message: 'Invalid JSON response for GET /vms' });
});
it.each([
  { ...vm, cpu: '2' }, { ...vm, cpu: 2.5 }, { ...vm, cpu: -1 },
  { ...vm, currentPowerState: undefined }, { ...vm, delete: 'false' },
  { ...vm, dataVolumeMessage: undefined }, { ...vm, phase: 'Impossible' },
])('rejects malformed nested VM fields %#', async malformedVm => {
  // Given a 2xx containing an invalid full VM rather than only invalid envelope.
  responseBody = JSON.stringify({ vms: [malformedVm] });
  // When the VM list loads.
  const result = createEndpoints(createApiClient(baseUrl)).vmApi.getVms();
  // Then no partial VM enters application state.
  await expect(result).rejects.toMatchObject({ name: 'ZodError' });
});
it.each([
  { ...adminConfig, sshGateway: { ...adminConfig.sshGateway, status: undefined } },
  { ...adminConfig, sshGateway: { ...adminConfig.sshGateway, status: { ...adminConfig.sshGateway.status, observedServiceName: 42 } } },
])('rejects missing or malformed admin gateway status %#', async config => {
  // Given an invalid nested administrator status.
  responseBody = JSON.stringify({ config });
  // When settings are parsed.
  const result = createEndpoints(createApiClient(baseUrl)).adminApi.getSettings();
  // Then required nested status and optional-string types are enforced.
  await expect(result).rejects.toMatchObject({ name: 'ZodError' });
});
it('accepts omitted observed strings and preserves explicit user status', async () => {
  // Given a valid Go settings object with omitted observed fields and an explicit user status.
  responseBody = JSON.stringify({ config: adminConfig, users: [{ ...user, status: 'Pending', password: 'private' }] });
  // When independent validated readers consume the fixture.
  const api = createEndpoints(createApiClient(baseUrl));
  const [settings, users] = await Promise.all([api.adminApi.getSettings(), api.adminApi.getUsers()]);
  // Then omission remains valid and private fields are stripped.
  expect(settings.config.sshGateway.status.observedServiceName).toBeUndefined();
  expect(users.users[0]).toEqual({ username: user.username, email: user.email, namespace: user.namespace, accessLevel: 1, status: 'Pending' });
});
