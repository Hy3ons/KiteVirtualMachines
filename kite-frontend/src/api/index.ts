import { apiClient } from './axios';
import { debugMockApi } from './debugMockStore';
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

const useMockApi = import.meta.env.VITE_USE_MOCK === 'true';

type BackendAdminUser = AdminUsersResponse['users'][number] & {
  readonly access_level?: number;
};

// ========================
// Auth
// ========================
export const authApi = {
  login: async (credentials: LoginCredentials) => {
    if (useMockApi) {
      return debugMockApi.login(credentials);
    }
    const { data } = await apiClient.post('/auth/login', credentials);
    return data;
  },
  signup: async (payload: SignupPayload) => {
    if (useMockApi) {
      return debugMockApi.signup(payload);
    }
    const { data } = await apiClient.post('/auth/signup', payload);
    return data;
  },
  logout: async (): Promise<LogoutResponse> => {
    if (useMockApi) {
      return debugMockApi.logout();
    }
    const { data } = await apiClient.post<LogoutResponse>('/auth/logout');
    return data;
  },
  getMe: async () => {
    if (useMockApi) {
      return debugMockApi.getMe();
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
      return debugMockApi.getConfig();
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
      return debugMockApi.getVms();
    }
    const { data } = await apiClient.get('/vms');
    return data;
  },
  getVm: async (name: string) => {
    if (useMockApi) {
      return debugMockApi.getVm(name);
    }
    const { data } = await apiClient.get(`/vms/${name}`);
    return data;
  },
  createVm: async (payload: CreateVmPayload) => {
    if (useMockApi) {
      return debugMockApi.createVm(payload);
    }
    const { data } = await apiClient.post('/vms', payload);
    return data;
  },
  updateVm: async (name: string, payload: UpdateVmPayload) => {
    if (useMockApi) {
      return debugMockApi.updateVm(name, payload);
    }
    const { data } = await apiClient.patch(`/vms/${name}`, payload);
    return data;
  },
  startVm: async (name: string) => {
    if (useMockApi) {
      return debugMockApi.startVm(name);
    }
    const { data } = await apiClient.post(`/vms/${name}/start`);
    return data;
  },
  stopVm: async (name: string) => {
    if (useMockApi) {
      return debugMockApi.stopVm(name);
    }
    const { data } = await apiClient.post(`/vms/${name}/stop`);
    return data;
  },
  deleteVm: async (name: string) => {
    if (useMockApi) {
      return debugMockApi.deleteVm(name);
    }
    const { data } = await apiClient.delete(`/vms/${name}`);
    return data;
  },
  createConsoleTicket: async (name: string) => {
    if (useMockApi) {
      return debugMockApi.createConsoleTicket(name);
    }
    const { data } = await apiClient.post(`/vms/${name}/console-ticket`);
    return data;
  },
  getOffers: async () => {
    if (useMockApi) {
      return debugMockApi.getOffers();
    }
    const { data } = await apiClient.get('/vm-offers');
    return data;
  },
  claimOffer: async (name: string, payload: ClaimVmOfferPayload) => {
    if (useMockApi) {
      return debugMockApi.claimOffer(name, payload);
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
      return debugMockApi.getUsers();
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
      return debugMockApi.updateUserAccess(nameOrUsername, accessLevel);
    }
    const { data } = await apiClient.patch(`/admin/users/${nameOrUsername}/access-level`, { access_level: accessLevel });
    return data;
  },
  deleteUser: async (nameOrUsername: string) => {
    if (useMockApi) {
      return debugMockApi.deleteUser(nameOrUsername);
    }
    const { data } = await apiClient.delete(`/admin/users/${nameOrUsername}`);
    return data;
  },
  getVms: async () => {
    if (useMockApi) {
      return debugMockApi.getAdminVms();
    }
    const { data } = await apiClient.get('/admin/vms');
    return data;
  },
  forceStopVm: async (namespace: string, name: string) => {
    if (useMockApi) {
      return debugMockApi.forceStopVm(namespace, name);
    }
    const { data } = await apiClient.patch(`/admin/vms/${namespace}/${name}/power`, { powerState: 'Off' });
    return data;
  },
  deleteVm: async (namespace: string, name: string) => {
    if (useMockApi) {
      return debugMockApi.deleteAdminVm(namespace, name);
    }
    const { data } = await apiClient.delete(`/admin/vms/${namespace}/${name}`);
    return data;
  },
  getSettings: async () => {
    if (useMockApi) {
      return debugMockApi.getSettings();
    }
    const { data } = await apiClient.get('/admin/settings');
    return data;
  },
  saveDomain: async (baseDomain: string) => {
    if (useMockApi) {
      return debugMockApi.saveDomain(baseDomain);
    }
    const { data } = await apiClient.post('/admin/domain', { baseDomain });
    return data;
  },
  saveHTTPSPolicy: async (payload: HTTPSPolicyPayload) => {
    if (useMockApi) {
      return debugMockApi.saveHTTPSPolicy(payload);
    }
    const { data } = await apiClient.post('/admin/https', payload);
    return data;
  },
  saveAdminContact: async (adminContact: string) => {
    if (useMockApi) {
      return debugMockApi.saveAdminContact(adminContact);
    }
    const { data } = await apiClient.post('/admin/admin-contact', { adminContact });
    return data;
  },
  createVmOffer: async (payload: AdminCreateVmOfferPayload) => {
    if (useMockApi) {
      return debugMockApi.createVmOffer(payload);
    }
    const { data } = await apiClient.post('/admin/vm-offers', payload);
    return data;
  },
  deleteVmOffer: async (namespace: string, name: string) => {
    if (useMockApi) {
      return debugMockApi.deleteVmOffer(namespace, name);
    }
    const { data } = await apiClient.delete(`/admin/vm-offers/${namespace}/${name}`);
    return data;
  },
  rotateRuntimeSecrets: async (payload: RuntimeSecretRotation) => {
    if (useMockApi) {
      return debugMockApi.rotateRuntimeSecrets(payload);
    }
    const { data } = await apiClient.post('/admin/runtime-secrets/rotate', payload);
    return data;
  },
  saveCert: async (payload: CertPayload) => {
    if (useMockApi) {
      return debugMockApi.saveCert(payload);
    }
    const { data } = await apiClient.post('/admin/cert', payload);
    return data;
  },
};
