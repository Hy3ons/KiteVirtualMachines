import type { AdminUser, ConfigResponse, UserVm } from './types';

export type DebugState = {
  readonly users: readonly AdminUser[];
  readonly vms: readonly UserVm[];
  readonly config: ConfigResponse['config'];
  readonly nextVmId: number;
};

export const DEBUG_STATE_KEY = 'kite_debug_mock_state';
export const DEFAULT_DEBUG_DOMAIN = 'apps.example.com';

export const defaultDebugState: DebugState = {
  users: [
    { username: 'admin', email: 'admin@kite.local', namespace: 'system', accessLevel: 3, status: 'Active' },
    { username: 'hyeonseok', email: 'hyeonseok@kite.local', namespace: 'hyeonseok-ns', accessLevel: 1, status: 'Active' },
    { username: 'viewer', email: 'viewer@kite.local', namespace: 'viewer-ns', accessLevel: 0, status: 'Pending' },
  ],
  vms: [
    {
      id: 'hyeonseok-ns/dev-vm-1',
      name: 'dev-vm-1',
      namespace: 'hyeonseok-ns',
      owner: 'hyeonseok-ns',
      domain: `dev.${DEFAULT_DEBUG_DOMAIN}`,
      phase: 'Running',
      powerState: 'On',
      currentPowerState: 'On',
      cpu: 2,
      memory: '4Gi',
      image: 'ubuntu-22.04',
      disk: '30Gi',
      domainPrefix: 'dev',
      sshId: 'ubuntu',
      delete: false,
      dataVolumePhase: 'Succeeded',
      dataVolumeProgress: '100.0%',
      dataVolumeMessage: 'Debug DataVolume is ready',
    },
    {
      id: 'hyeonseok-ns/test-db',
      name: 'test-db',
      namespace: 'hyeonseok-ns',
      owner: 'hyeonseok-ns',
      domain: `db.${DEFAULT_DEBUG_DOMAIN}`,
      phase: 'Stopped',
      powerState: 'Off',
      currentPowerState: 'Off',
      cpu: 2,
      memory: '4Gi',
      image: 'ubuntu-22.04',
      disk: '50Gi',
      domainPrefix: 'db',
      sshId: 'dbuser',
      delete: false,
      dataVolumePhase: 'Succeeded',
      dataVolumeProgress: '100.0%',
      dataVolumeMessage: 'Debug DataVolume is ready',
    },
  ],
  config: {
    baseDomain: DEFAULT_DEBUG_DOMAIN,
    hasJWTSecret: true,
    hasPasswordSalt: true,
    hasTLSCertificate: true,
  },
  nextVmId: 3,
};
