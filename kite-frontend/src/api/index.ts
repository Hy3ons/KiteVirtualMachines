import { apiClient } from './axios';
import type {
  AdminUsersResponse,
  AdminCreateVmOfferPayload,
  CertPayload,
  ClaimVmOfferPayload,
  CreateVmPayload,
  HTTPSPolicyPayload,
  LoginCredentials,
  LogoutResponse,
  RuntimeSecretRotation,
  SignupPayload,
  UpdateVmPayload,
} from './types';

class MockApiModeError extends Error {
  constructor(mode: string) {
    super(`VITE_USE_MOCK=true is only allowed in debug mode, current mode is ${mode}`);
    this.name = 'MockApiModeError';
  }
}

const isDebugMode = import.meta.env.MODE === 'debug';
const mockApiRequested = import.meta.env.VITE_USE_MOCK === 'true';

if (mockApiRequested && !isDebugMode) {
  throw new MockApiModeError(import.meta.env.MODE);
}

const useMockApi = mockApiRequested && isDebugMode;

type DebugMockApi = typeof import('./debugMockStore')['debugMockApi'];

let debugMockApiPromise: Promise<DebugMockApi> | null = null;

const loadDebugMockApi = async (): Promise<DebugMockApi> => {
  debugMockApiPromise ??= import('./debugMockStore').then(({ debugMockApi }) => debugMockApi);
  return debugMockApiPromise;
};

type BackendAdminUser = AdminUsersResponse['users'][number] & {
  readonly access_level?: number;
};

// ========================
// Auth
// ========================
export const authApi = {
  login: async (credentials: LoginCredentials) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).login(credentials);
    }
    const { data } = await apiClient.post('/auth/login', credentials);
    return data;
  },
  signup: async (payload: SignupPayload) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).signup(payload);
    }
    const { data } = await apiClient.post('/auth/signup', payload);
    return data;
  },
  logout: async (): Promise<LogoutResponse> => {
    if (useMockApi) {
      return (await loadDebugMockApi()).logout();
    }
    const { data } = await apiClient.post<LogoutResponse>('/auth/logout');
    return data;
  },
  getMe: async () => {
    if (useMockApi) {
      return (await loadDebugMockApi()).getMe();
    }
    const { data } = await apiClient.get('/me');
    return data;
  },
};

// ========================
// Config
// ========================
export const configApi = {
  getConfig: async () => {
    if (useMockApi) {
      return (await loadDebugMockApi()).getConfig();
    }
    const { data } = await apiClient.get('/config');
    return data;
  },
};

// ========================
// User VMs
// ========================
export const vmApi = {
  getVms: async () => {
    if (useMockApi) {
      return (await loadDebugMockApi()).getVms();
    }
    const { data } = await apiClient.get('/vms');
    return data;
  },
  getVm: async (name: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).getVm(name);
    }
    const { data } = await apiClient.get(`/vms/${name}`);
    return data;
  },
  createVm: async (payload: CreateVmPayload) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).createVm(payload);
    }
    const { data } = await apiClient.post('/vms', payload);
    return data;
  },
  updateVm: async (name: string, payload: UpdateVmPayload) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).updateVm(name, payload);
    }
    const { data } = await apiClient.patch(`/vms/${name}`, payload);
    return data;
  },
  startVm: async (name: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).startVm(name);
    }
    const { data } = await apiClient.post(`/vms/${name}/start`);
    return data;
  },
  stopVm: async (name: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).stopVm(name);
    }
    const { data } = await apiClient.post(`/vms/${name}/stop`);
    return data;
  },
  deleteVm: async (name: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).deleteVm(name);
    }
    const { data } = await apiClient.delete(`/vms/${name}`);
    return data;
  },
  createConsoleTicket: async (name: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).createConsoleTicket(name);
    }
    const { data } = await apiClient.post(`/vms/${name}/console-ticket`);
    return data;
  },
  getOffers: async () => {
    if (useMockApi) {
      return (await loadDebugMockApi()).getOffers();
    }
    const { data } = await apiClient.get('/vm-offers');
    return data;
  },
  claimOffer: async (name: string, payload: ClaimVmOfferPayload) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).claimOffer(name, payload);
    }
    const { data } = await apiClient.post(`/vm-offers/${name}/claim`, payload);
    return data;
  },
};

// ========================
// Admin
// ========================
export const adminApi = {
  getUsers: async () => {
    if (useMockApi) {
      return (await loadDebugMockApi()).getUsers();
    }
    const { data } = await apiClient.get('/admin/users');
    if (data && data.users) {
      data.users = data.users.map((u: BackendAdminUser) => ({
        ...u,
        accessLevel: u.access_level
      }));
    }
    return data;
  },
  updateUserAccess: async (nameOrUsername: string, accessLevel: number) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).updateUserAccess(nameOrUsername, accessLevel);
    }
    const { data } = await apiClient.patch(`/admin/users/${nameOrUsername}/access-level`, { access_level: accessLevel });
    return data;
  },
  deleteUser: async (nameOrUsername: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).deleteUser(nameOrUsername);
    }
    const { data } = await apiClient.delete(`/admin/users/${nameOrUsername}`);
    return data;
  },
  getVms: async () => {
    if (useMockApi) {
      return (await loadDebugMockApi()).getAdminVms();
    }
    const { data } = await apiClient.get('/admin/vms');
    return data;
  },
  forceStopVm: async (namespace: string, name: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).forceStopVm(namespace, name);
    }
    const { data } = await apiClient.patch(`/admin/vms/${namespace}/${name}/power`, { powerState: 'Off' });
    return data;
  },
  deleteVm: async (namespace: string, name: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).deleteAdminVm(namespace, name);
    }
    const { data } = await apiClient.delete(`/admin/vms/${namespace}/${name}`);
    return data;
  },
  getSettings: async () => {
    if (useMockApi) {
      return (await loadDebugMockApi()).getSettings();
    }
    const { data } = await apiClient.get('/admin/settings');
    return data;
  },
  saveDomain: async (baseDomain: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).saveDomain(baseDomain);
    }
    const { data } = await apiClient.post('/admin/domain', { baseDomain });
    return data;
  },
  saveHTTPSPolicy: async (payload: HTTPSPolicyPayload) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).saveHTTPSPolicy(payload);
    }
    const { data } = await apiClient.post('/admin/https', payload);
    return data;
  },
  saveAdminContact: async (adminContact: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).saveAdminContact(adminContact);
    }
    const { data } = await apiClient.post('/admin/admin-contact', { adminContact });
    return data;
  },
  createVmOffer: async (payload: AdminCreateVmOfferPayload) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).createVmOffer(payload);
    }
    const { data } = await apiClient.post('/admin/vm-offers', payload);
    return data;
  },
  deleteVmOffer: async (namespace: string, name: string) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).deleteVmOffer(namespace, name);
    }
    const { data } = await apiClient.delete(`/admin/vm-offers/${namespace}/${name}`);
    return data;
  },
  rotateRuntimeSecrets: async (payload: RuntimeSecretRotation) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).rotateRuntimeSecrets(payload);
    }
    const { data } = await apiClient.post('/admin/runtime-secrets/rotate', payload);
    return data;
  },
  saveCert: async (payload: CertPayload) => {
    if (useMockApi) {
      return (await loadDebugMockApi()).saveCert(payload);
    }
    const { data } = await apiClient.post('/admin/cert', payload);
    return data;
  },
};
