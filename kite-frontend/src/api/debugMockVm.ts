import { makeSyntheticVm, readDebugState, toDiskString, writeDebugState } from './debugMockState';
import type { CreateVmPayload, UpdateVmPayload, UserVm, VmResponse, VmsResponse } from './types';

const replaceVm = (vms: readonly UserVm[], name: string, vm: UserVm): readonly UserVm[] => [
  ...vms.filter((candidate) => candidate.name !== name),
  vm,
];

const withPowerState = (vm: UserVm, powerState: 'On' | 'Off'): UserVm => ({
  ...vm,
  powerState,
  currentPowerState: powerState,
  phase: powerState === 'On' ? 'Running' : 'Stopped',
});

export const debugVmApi = {
  getVms: async (): Promise<VmsResponse> => ({ vms: readDebugState().vms.filter((vm) => !vm.delete) }),

  getVm: async (name: string): Promise<VmResponse> => {
    const state = readDebugState();
    const existing = state.vms.find((vm) => vm.name === name && !vm.delete);
    if (existing) {
      return { vm: existing };
    }

    const vm = makeSyntheticVm(name, state);
    writeDebugState({ ...state, vms: [...state.vms, vm] });
    return { vm };
  },

  createVm: async (payload: CreateVmPayload): Promise<VmResponse> => {
    const state = readDebugState();
    const domainPrefix = payload.domainPrefix || payload.name;
    const powerState = payload.powerState || 'On';
    const vm: UserVm = {
      id: `hyeonseok-ns/${payload.name}`,
      name: payload.name,
      namespace: 'hyeonseok-ns',
      owner: 'hyeonseok-ns',
      domain: `${domainPrefix}.${state.config.baseDomain}`,
      phase: powerState === 'On' ? 'Running' : 'Stopped',
      powerState,
      currentPowerState: powerState,
      cpu: payload.cpu || 2,
      memory: payload.memory || '4Gi',
      image: 'ubuntu-22.04',
      disk: toDiskString(payload.disk),
      domainPrefix,
      sshId: payload.sshId,
      delete: false,
      dataVolumePhase: 'Succeeded',
      dataVolumeProgress: '100.0%',
      dataVolumeMessage: 'Debug VM created in local browser state',
    };

    writeDebugState({
      ...state,
      nextVmId: state.nextVmId + 1,
      vms: replaceVm(state.vms, payload.name, vm),
    });

    return { vm };
  },

  updateVm: async (name: string, payload: UpdateVmPayload): Promise<VmResponse> => {
    const state = readDebugState();
    const vm = state.vms.find((candidate) => candidate.name === name) || makeSyntheticVm(name, state);
    const domainPrefix = payload.domainPrefix || vm.domainPrefix;
    const powerState = payload.powerState || vm.powerState;
    const updated: UserVm = {
      ...withPowerState(vm, powerState),
      domainPrefix,
      domain: `${domainPrefix}.${state.config.baseDomain}`,
    };

    writeDebugState({ ...state, vms: replaceVm(state.vms, name, updated) });
    return { vm: updated };
  },

  startVm: async (name: string): Promise<VmResponse> => debugVmApi.updateVm(name, { powerState: 'On' }),

  stopVm: async (name: string): Promise<VmResponse> => debugVmApi.updateVm(name, { powerState: 'Off' }),

  deleteVm: async (name: string): Promise<VmResponse> => {
    const state = readDebugState();
    const vm = state.vms.find((candidate) => candidate.name === name) || makeSyntheticVm(name, state);
    const deleted: UserVm = { ...vm, delete: true, phase: 'Terminating' };
    writeDebugState({ ...state, vms: replaceVm(state.vms, name, deleted) });
    return { vm: deleted };
  },
};
