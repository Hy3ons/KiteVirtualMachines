import { DEBUG_STATE_KEY, defaultDebugState, type DebugState } from './debugMockData';
import type { GlobalVm, UserVm } from './types';

const hasKey = <Key extends string>(value: object, key: Key): value is Record<Key, unknown> => key in value;

const isDebugState = (value: unknown): value is DebugState => {
  if (typeof value !== 'object' || value === null) {
    return false;
  }

  return (
    hasKey(value, 'users') &&
    hasKey(value, 'vms') &&
    hasKey(value, 'config') &&
    hasKey(value, 'nextVmId') &&
    Array.isArray(value.users) &&
    Array.isArray(value.vms) &&
    typeof value.config === 'object' &&
    value.config !== null &&
    typeof value.nextVmId === 'number'
  );
};

const getBrowserStorage = (): Storage | null => {
  if (typeof window === 'undefined') {
    return null;
  }

  return window.localStorage;
};

export const readDebugState = (): DebugState => {
  const storage = getBrowserStorage();
  if (!storage) {
    return defaultDebugState;
  }

  const rawState = storage.getItem(DEBUG_STATE_KEY);
  if (!rawState) {
    storage.setItem(DEBUG_STATE_KEY, JSON.stringify(defaultDebugState));
    return defaultDebugState;
  }

  try {
    const parsed: unknown = JSON.parse(rawState);
    if (isDebugState(parsed)) {
      return parsed;
    }
  } catch (error) {
    if (error instanceof SyntaxError) {
      storage.removeItem(DEBUG_STATE_KEY);
      storage.setItem(DEBUG_STATE_KEY, JSON.stringify(defaultDebugState));
      return defaultDebugState;
    }

    throw error;
  }

  storage.setItem(DEBUG_STATE_KEY, JSON.stringify(defaultDebugState));
  return defaultDebugState;
};

export const writeDebugState = (state: DebugState): DebugState => {
  const storage = getBrowserStorage();
  if (storage) {
    storage.setItem(DEBUG_STATE_KEY, JSON.stringify(state));
  }

  return state;
};

export const toDiskString = (disk: number | string): string => {
  if (typeof disk === 'number') {
    return `${disk}Gi`;
  }

  return disk.endsWith('Gi') ? disk : `${disk}Gi`;
};

export const makeSyntheticVm = (name: string, state: DebugState): UserVm => {
  const domainPrefix = name.replace(/[^a-z0-9-]/gi, '-').toLowerCase();
  return {
    id: `hyeonseok-ns/${name}`,
    name,
    namespace: 'hyeonseok-ns',
    owner: 'hyeonseok-ns',
    domain: `${domainPrefix}.${state.config.baseDomain}`,
    phase: 'Stopped',
    powerState: 'Off',
    currentPowerState: 'Off',
    cpu: 2,
    memory: '4Gi',
    image: 'ubuntu-22.04',
    disk: '20Gi',
    domainPrefix,
    sshId: 'ubuntu',
    delete: false,
    dataVolumePhase: 'Succeeded',
    dataVolumeProgress: '100.0%',
    dataVolumeMessage: 'Synthetic debug VM for direct route QA',
  };
};

export const toGlobalVm = (vm: UserVm): GlobalVm => ({
  id: vm.id,
  owner: vm.owner,
  namespace: vm.namespace,
  name: vm.name,
  domain: vm.domain,
  phase: vm.phase,
  cpu: vm.cpu,
  memory: vm.memory,
});
