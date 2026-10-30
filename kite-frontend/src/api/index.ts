import { createEndpoints } from './endpoints';
import { apiClient } from './http';
import * as contracts from './contracts';
import * as settings from './configContracts';
class MockApiModeError extends Error {
  constructor(mode: string) {
    super(`NEXT_PUBLIC_USE_MOCK=true is only allowed in debug mode, current mode is ${mode}`);
    this.name = 'MockApiModeError';
  }
}

const isDebugMode = process.env.NEXT_PUBLIC_KITE_MODE === 'debug';
const mockApiRequested = process.env.NEXT_PUBLIC_USE_MOCK === 'true';

if (mockApiRequested && !isDebugMode) {
  throw new MockApiModeError(process.env.NEXT_PUBLIC_KITE_MODE || 'production');
}

const useMockApi = mockApiRequested && isDebugMode;

type DebugMockApi = typeof import('./debugMockStore')['debugMockApi'];

let debugMockApiPromise: Promise<DebugMockApi> | null = null;

const loadDebugMockApi = async (): Promise<DebugMockApi> => {
  debugMockApiPromise ??= import('./debugMockStore').then(({ debugMockApi }) => debugMockApi);
  return debugMockApiPromise;
};


const endpoints = createEndpoints(apiClient);
export const authApi = {
  login: async (...args: Parameters<typeof endpoints.authApi.login>) => {
    if (useMockApi) return contracts.authResponseSchema.parse(await (await loadDebugMockApi()).login(...args));
    return endpoints.authApi.login(...args);
  },
  signup: async (...args: Parameters<typeof endpoints.authApi.signup>) => {
    if (useMockApi) return contracts.signupResponseSchema.parse(await (await loadDebugMockApi()).signup(...args));
    return endpoints.authApi.signup(...args);
  },
  logout: async (...args: Parameters<typeof endpoints.authApi.logout>) => {
    if (useMockApi) return contracts.messageSchema.parse(await (await loadDebugMockApi()).logout(...args));
    return endpoints.authApi.logout(...args);
  },
  getMe: async (...args: Parameters<typeof endpoints.authApi.getMe>) => {
    if (useMockApi) return contracts.meSchema.parse(await (await loadDebugMockApi()).getMe(...args));
    return endpoints.authApi.getMe(...args);
  },
};
export const configApi = {
  getConfig: async (...args: Parameters<typeof endpoints.configApi.getConfig>) => {
    if (useMockApi) return settings.publicConfigSchema.parse(await (await loadDebugMockApi()).getConfig(...args));
    return endpoints.configApi.getConfig(...args);
  },
};
export const vmApi = {
  getVms: async (...args: Parameters<typeof endpoints.vmApi.getVms>) => {
    if (useMockApi) return contracts.vmsSchema.parse(await (await loadDebugMockApi()).getVms(...args));
    return endpoints.vmApi.getVms(...args);
  },
  getVm: async (...args: Parameters<typeof endpoints.vmApi.getVm>) => {
    if (useMockApi) return contracts.vmResponseSchema.parse(await (await loadDebugMockApi()).getVm(...args));
    return endpoints.vmApi.getVm(...args);
  },
  createVm: async (...args: Parameters<typeof endpoints.vmApi.createVm>) => {
    if (useMockApi) return contracts.vmResponseSchema.parse(await (await loadDebugMockApi()).createVm(...args));
    return endpoints.vmApi.createVm(...args);
  },
  updateVm: async (...args: Parameters<typeof endpoints.vmApi.updateVm>) => {
    if (useMockApi) return contracts.vmResponseSchema.parse(await (await loadDebugMockApi()).updateVm(...args));
    return endpoints.vmApi.updateVm(...args);
  },
  startVm: async (...args: Parameters<typeof endpoints.vmApi.startVm>) => {
    if (useMockApi) return contracts.vmResponseSchema.parse(await (await loadDebugMockApi()).startVm(...args));
    return endpoints.vmApi.startVm(...args);
  },
  stopVm: async (...args: Parameters<typeof endpoints.vmApi.stopVm>) => {
    if (useMockApi) return contracts.vmResponseSchema.parse(await (await loadDebugMockApi()).stopVm(...args));
    return endpoints.vmApi.stopVm(...args);
  },
  deleteVm: async (...args: Parameters<typeof endpoints.vmApi.deleteVm>) => {
    if (useMockApi) return contracts.vmDeletionSchema.parse(await (await loadDebugMockApi()).deleteVm(...args));
    return endpoints.vmApi.deleteVm(...args);
  },
  createConsoleTicket: async (...args: Parameters<typeof endpoints.vmApi.createConsoleTicket>) => {
    if (useMockApi) return contracts.consoleTicketSchema.parse(await (await loadDebugMockApi()).createConsoleTicket(...args));
    return endpoints.vmApi.createConsoleTicket(...args);
  },
  getOffers: async (...args: Parameters<typeof endpoints.vmApi.getOffers>) => {
    if (useMockApi) return contracts.offersSchema.parse(await (await loadDebugMockApi()).getOffers(...args));
    return endpoints.vmApi.getOffers(...args);
  },
  claimOffer: async (...args: Parameters<typeof endpoints.vmApi.claimOffer>) => {
    if (useMockApi) return contracts.vmResponseSchema.parse(await (await loadDebugMockApi()).claimOffer(...args));
    return endpoints.vmApi.claimOffer(...args);
  },
};
export const adminApi = {
  getUsers: async (...args: Parameters<typeof endpoints.adminApi.getUsers>) => {
    if (useMockApi) return contracts.usersSchema.parse(await (await loadDebugMockApi()).getUsers(...args));
    return endpoints.adminApi.getUsers(...args);
  },
  updateUserAccess: async (...args: Parameters<typeof endpoints.adminApi.updateUserAccess>) => {
    if (useMockApi) return contracts.meSchema.parse(await (await loadDebugMockApi()).updateUserAccess(...args));
    return endpoints.adminApi.updateUserAccess(...args);
  },
  deleteUser: async (...args: Parameters<typeof endpoints.adminApi.deleteUser>) => {
    if (useMockApi) return contracts.messageSchema.parse(await (await loadDebugMockApi()).deleteUser(...args));
    return endpoints.adminApi.deleteUser(...args);
  },
  getVms: async (...args: Parameters<typeof endpoints.adminApi.getVms>) => {
    if (useMockApi) return contracts.globalVmsSchema.parse(await (await loadDebugMockApi()).getAdminVms(...args));
    return endpoints.adminApi.getVms(...args);
  },
  forceStopVm: async (...args: Parameters<typeof endpoints.adminApi.forceStopVm>) => {
    if (useMockApi) return contracts.vmResponseSchema.parse(await (await loadDebugMockApi()).forceStopVm(...args));
    return endpoints.adminApi.forceStopVm(...args);
  },
  deleteVm: async (...args: Parameters<typeof endpoints.adminApi.deleteVm>) => {
    if (useMockApi) return contracts.vmDeletionSchema.parse(await (await loadDebugMockApi()).deleteAdminVm(...args));
    return endpoints.adminApi.deleteVm(...args);
  },
  getSettings: async (...args: Parameters<typeof endpoints.adminApi.getSettings>) => {
    if (useMockApi) return settings.adminConfigSchema.parse(await (await loadDebugMockApi()).getSettings(...args));
    return endpoints.adminApi.getSettings(...args);
  },
  saveDomain: async (...args: Parameters<typeof endpoints.adminApi.saveDomain>) => {
    if (useMockApi) return settings.settingsMutationSchema.parse(await (await loadDebugMockApi()).saveDomain(...args));
    return endpoints.adminApi.saveDomain(...args);
  },
  saveHTTPSPolicy: async (...args: Parameters<typeof endpoints.adminApi.saveHTTPSPolicy>) => {
    if (useMockApi) return settings.settingsMutationSchema.parse(await (await loadDebugMockApi()).saveHTTPSPolicy(...args));
    return endpoints.adminApi.saveHTTPSPolicy(...args);
  },
  saveSSHGateway: async (...args: Parameters<typeof endpoints.adminApi.saveSSHGateway>) => {
    if (useMockApi) return settings.settingsMutationSchema.parse(await (await loadDebugMockApi()).saveSSHGateway(...args));
    return endpoints.adminApi.saveSSHGateway(...args);
  },
  saveAdminContact: async (...args: Parameters<typeof endpoints.adminApi.saveAdminContact>) => {
    if (useMockApi) return settings.settingsMutationSchema.parse(await (await loadDebugMockApi()).saveAdminContact(...args));
    return endpoints.adminApi.saveAdminContact(...args);
  },
  createVmOffer: async (...args: Parameters<typeof endpoints.adminApi.createVmOffer>) => {
    if (useMockApi) return contracts.offerResponseSchema.parse(await (await loadDebugMockApi()).createVmOffer(...args));
    return endpoints.adminApi.createVmOffer(...args);
  },
  deleteVmOffer: async (...args: Parameters<typeof endpoints.adminApi.deleteVmOffer>) => {
    if (useMockApi) return contracts.messageSchema.parse(await (await loadDebugMockApi()).deleteVmOffer(...args));
    return endpoints.adminApi.deleteVmOffer(...args);
  },
  rotateRuntimeSecrets: async (...args: Parameters<typeof endpoints.adminApi.rotateRuntimeSecrets>) => {
    if (useMockApi) return settings.settingsMutationSchema.parse(await (await loadDebugMockApi()).rotateRuntimeSecrets(...args));
    return endpoints.adminApi.rotateRuntimeSecrets(...args);
  },
  saveCert: async (...args: Parameters<typeof endpoints.adminApi.saveCert>) => {
    if (useMockApi) return settings.settingsMutationSchema.parse(await (await loadDebugMockApi()).saveCert(...args));
    return endpoints.adminApi.saveCert(...args);
  },
};
