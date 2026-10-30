import { describe, expect, it } from 'vitest';
import { buildVmCreatePayload, canControlVm, canManageSystem, canManageUsers, creationPolicy } from '../src/domain/access';
import { validTCPPort } from '../src/domain/network';
import { vmLifecycle } from '../src/domain/vmLifecycle';
import { buildSSHCommandState } from '../src/screens/sshGatewayCommand';
import type { AccessLevel, SSHGatewaySettings, VmPhase } from '../src/api/types';

const roles: readonly (AccessLevel | null)[] = [null, 0, 1, 2, 3];

describe('workspace access policy', () => {
  it.each(roles)('fails closed and separates privileges for role %s', (role) => {
    // Given an authenticated role or an unavailable session.
    // When workspace capabilities are derived.
    const capabilities = [canControlVm(role), canManageUsers(role), canManageSystem(role)];
    // Then only the authorized role levels gain each capability.
    expect(capabilities).toEqual([role !== null && role >= 1, role !== null && role >= 2, role === 3]);
  });
  it.each([0, 1, 2, 3])('counts inventory entries including deletion-pending entries at count %i', (count) => {
    // Given the complete inventory length, without filtering deletion-pending entries.
    // When a normal account requests creation.
    const result = creationPolicy(1, count);
    // Then the second occupied slot prevents creation until removal completes.
    expect(result).toBe(count < 2 ? 'allowed' : 'quota');
  });
  it.each(roles)('applies creation access before quota for role %s', (role) => {
    // Given an inventory above the normal quota.
    // When creation is requested.
    const result = creationPolicy(role, 5);
    // Then elevated accounts retain free creation and missing access fails closed.
    expect(result).toBe(role === null || role === 0 ? 'forbidden' : role === 1 ? 'quota' : 'allowed');
  });
});

const form = { name: 'demo', domainPrefix: 'demo', sshId: 'ubuntu', sshPassword: 'secret', disk: 80, cpu: 8, memory: '16Gi' };
it('overrides requested resources for the fixed-spec role', () => {
  // Given a normal account requesting larger resources.
  // When its payload is built.
  const payload = buildVmCreatePayload(1, form);
  // Then the enforced spec and credentials survive as the API payload.
  expect(payload).toEqual({ ...form, cpu: 2, memory: '4Gi', disk: 25, powerState: 'On' });
});
it.each([2, 3] satisfies AccessLevel[])('preserves elevated resources for role %i', (role) => {
  // Given an elevated account and valid custom resources.
  // When its payload is built.
  const payload = buildVmCreatePayload(role, form);
  // Then the requested resources are preserved.
  expect(payload).toEqual({ ...form, powerState: 'On' });
});
it('enforces the minimum disk for free-spec creation', () => {
  // Given a free-spec form below the storage minimum.
  // When its payload is built.
  const payload = buildVmCreatePayload(2, { ...form, disk: 1 });
  // Then storage never falls below the platform minimum.
  expect(payload.disk).toBe(20);
});
const phases: readonly VmPhase[] = ['Running', 'Stopped', 'Creating', 'Terminating', 'Provisioning', 'Ready', 'Deleting', 'Failed'];
it.each(phases)('derives connection and power actions for %s', (phase) => {
  // Given observed VM phase without a deletion request.
  // When lifecycle policy is calculated.
  const policy = vmLifecycle({ phase, delete: false });
  // Then only Running connects and every other phase requests power On.
  expect(policy).toEqual({ running: phase === 'Running', canConnect: phase === 'Running', canMutate: true, nextPower: phase === 'Running' ? 'Off' : 'On' });
});
it.each(phases)('blocks connection and mutation during deletion for %s', (phase) => {
  // Given deletion pending in any observed phase.
  // When lifecycle policy is calculated.
  const policy = vmLifecycle({ phase, delete: true });
  // Then no connection or mutation is offered.
  expect([policy.canConnect, policy.canMutate]).toEqual([false, false]);
});
it.each(['1', '22', '65535', ' 2222 '])('accepts decimal TCP port %s', (port) => {
  // Given a supported decimal input. When it is validated. Then it is accepted.
  expect(validTCPPort(port)).toBe(true);
});
it.each(['', ' ', '0', '65536', '-1', '2.5', '1e2', '0x16', 'abc'])('rejects invalid TCP port %s', (port) => {
  // Given an unsupported input. When it is validated. Then it is rejected.
  expect(validTCPPort(port)).toBe(false);
});
const gateway: SSHGatewaySettings = { externalEnabled: true, externalPort: '2222', publicPort: '22', status: { phase: 'Ready', reason: '', message: '' } };
it('does not expose a command when platform configuration is unavailable', () => {
  // Given a failed optional configuration fetch.
  // When connection instructions are derived.
  const result = buildSSHCommandState('ubuntu', 'example.test', null);
  // Then no executable command is offered.
  expect(result.ready).toBe(false);
  expect('command' in result).toBe(false);
});
it.each(['22', '2222'])('uses the published SSH port %s', (port) => {
  // Given a ready gateway with a published port distinct from its Service port.
  // When instructions are built.
  const result = buildSSHCommandState('ubuntu', 'example.test', { ...gateway, publicPort: port });
  // Then the ready variant carries the exact executable command.
  expect(result).toEqual({ ready: true, command: port === '22' ? 'ssh ubuntu@example.test' : 'ssh -p 2222 ubuntu@example.test', configPort: port });
});
it.each(['', '0', '65536', '1e2'])('withholds SSH command for invalid published port %s', (port) => {
  // Given a ready gateway whose advertised and Service ports are invalid.
  // When connection instructions are built.
  const result = buildSSHCommandState('ubuntu', 'example.test', { ...gateway, publicPort: port, externalPort: port });
  // Then the unavailable variant cannot carry a command.
  expect(result.ready).toBe(false);
  expect('command' in result).toBe(false);
});
it('withholds SSH command for a disabled gateway', () => {
  // Given an externally disabled gateway that last reported Ready.
  // When connection instructions are built.
  const result = buildSSHCommandState('ubuntu', 'example.test', { ...gateway, externalEnabled: false });
  // Then stale observed readiness cannot override operator intent.
  expect(result.ready).toBe(false);
});
it('withholds SSH command when the base domain is missing', () => {
  // Given a ready gateway without an advertised domain.
  // When connection instructions are built.
  const result = buildSSHCommandState('ubuntu', ' ', gateway);
  // Then there is no executable destination.
  expect(result.ready).toBe(false);
});
