import { readDebugState, toGlobalVm, writeDebugState } from './debugMockState';
import { debugVmApi } from './debugMockVm';
import type {
  AdminUsersResponse,
  CertPayload,
  ConfigResponse,
  GlobalVmsResponse,
  HTTPSPolicyPayload,
  RuntimeSecretRotation,
  UserVm,
  VmResponse,
} from './types';

export const debugAdminApi = {
  getUsers: async (): Promise<AdminUsersResponse> => ({ users: readDebugState().users }),

  updateUserAccess: async (nameOrUsername: string, accessLevel: number): Promise<AdminUsersResponse> => {
    const state = readDebugState();
    const users = state.users.map((user) =>
      user.username === nameOrUsername ? { ...user, accessLevel, status: 'Active' as const } : user
    );
    writeDebugState({ ...state, users });
    return { users };
  },

  deleteUser: async (nameOrUsername: string): Promise<AdminUsersResponse> => {
    const state = readDebugState();
    const user = state.users.find((candidate) => candidate.username === nameOrUsername);
    const namespace = user?.namespace || nameOrUsername;
    const users = state.users.filter((candidate) => candidate.username !== nameOrUsername);
    const vms = state.vms.filter((vm) => vm.namespace !== namespace);
    writeDebugState({ ...state, users, vms });
    return { users };
  },

  getAdminVms: async (): Promise<GlobalVmsResponse> => ({
    vms: readDebugState().vms.filter((vm) => !vm.delete).map(toGlobalVm),
  }),

  forceStopVm: async (namespace: string, name: string): Promise<VmResponse> => {
    const state = readDebugState();
    const vm = state.vms.find((candidate) => candidate.namespace === namespace && candidate.name === name);
    if (!vm) {
      return debugVmApi.stopVm(name);
    }

    const stopped: UserVm = { ...vm, powerState: 'Off', currentPowerState: 'Off', phase: 'Stopped' };
    writeDebugState({
      ...state,
      vms: [...state.vms.filter((candidate) => candidate.id !== vm.id), stopped],
    });
    return { vm: stopped };
  },

  deleteAdminVm: async (namespace: string, name: string): Promise<VmResponse> => {
    const state = readDebugState();
    const vm = state.vms.find((candidate) => candidate.namespace === namespace && candidate.name === name);
    if (!vm) {
      return debugVmApi.deleteVm(name);
    }

    const deleted: UserVm = { ...vm, delete: true, phase: 'Terminating' };
    writeDebugState({
      ...state,
      vms: [...state.vms.filter((candidate) => candidate.id !== vm.id), deleted],
    });
    return { vm: deleted };
  },

  getSettings: async (): Promise<ConfigResponse> => ({ config: readDebugState().config }),

  saveDomain: async (baseDomain: string): Promise<ConfigResponse> => {
    const state = readDebugState();
    const config = { ...state.config, baseDomain };
    const vms = state.vms.map((vm) => ({ ...vm, domain: `${vm.domainPrefix}.${baseDomain}` }));
    writeDebugState({ ...state, config, vms });
    return { config };
  },

  saveHTTPSPolicy: async (payload: HTTPSPolicyPayload): Promise<ConfigResponse> => {
    const state = readDebugState();
    const config = { ...state.config, forceHttps: payload.forceHttps };
    writeDebugState({ ...state, config });
    return { config };
  },

  rotateRuntimeSecrets: async (payload: RuntimeSecretRotation): Promise<ConfigResponse> => {
    const state = readDebugState();
    const config = {
      ...state.config,
      hasJWTSecret: payload.rotateJWTSecret ? true : state.config.hasJWTSecret,
      hasPasswordSalt: payload.rotatePasswordSalt ? true : state.config.hasPasswordSalt,
    };
    writeDebugState({ ...state, config });
    return { config };
  },

  saveCert: async (payload: CertPayload): Promise<ConfigResponse> => {
    const state = readDebugState();
    const config = { ...state.config, hasTLSCertificate: payload.tlsCert.length > 0 && payload.tlsKey.length > 0 };
    writeDebugState({ ...state, config });
    return { config };
  },
};
