import * as contracts from './contracts';
import * as settings from './configContracts';
import type { ApiClient } from './client';
import type {
  AdminCreateVmOfferPayload,
  CertPayload,
  ClaimVmOfferPayload,
  CreateVmPayload,
  HTTPSPolicyPayload,
  LoginCredentials,
  RuntimeSecretRotation,
  SSHGatewayPayload,
  SignupPayload,
  UpdateVmPayload,
  AccessLevel,
} from './types';

// createEndpoints binds validated Go operations to a browser or request-scoped SSR transport.
export function createEndpoints(apiClient: ApiClient) {
  const authApi = {
    login: async (credentials: LoginCredentials) => {
      const { data } = await apiClient.post('/auth/login', credentials);
      return contracts.authResponseSchema.parse(data);
    },
    signup: async (payload: SignupPayload) => {
      const { data } = await apiClient.post('/auth/signup', payload);
      return contracts.signupResponseSchema.parse(data);
    },
    logout: async () => {
      const { data } = await apiClient.post('/auth/logout');
      return contracts.messageSchema.parse(data);
    },
    getMe: async () => {
      const { data } = await apiClient.get('/me');
      return contracts.sessionResponseSchema.parse(data);
    },
  };
  const configApi = {
    getConfig: async () => {
      const { data } = await apiClient.get('/config');
      return settings.publicConfigSchema.parse(data);
    },
  };
  const vmApi = {
    getVms: async () => {
      const { data } = await apiClient.get('/vms');
      return contracts.vmsSchema.parse(data);
    },
    getVm: async (name: string) => {
      const { data } = await apiClient.get(`/vms/${encodeURIComponent(name)}`);
      return contracts.vmResponseSchema.parse(data);
    },
    createVm: async (payload: CreateVmPayload) => {
      const { data } = await apiClient.post('/vms', payload);
      return contracts.vmResponseSchema.parse(data);
    },
    updateVm: async (name: string, payload: UpdateVmPayload) => {
      const { data } = await apiClient.patch(`/vms/${encodeURIComponent(name)}`, payload);
      return contracts.vmResponseSchema.parse(data);
    },
    startVm: async (name: string) => {
      const { data } = await apiClient.post(`/vms/${encodeURIComponent(name)}/start`);
      return contracts.vmResponseSchema.parse(data);
    },
    stopVm: async (name: string) => {
      const { data } = await apiClient.post(`/vms/${encodeURIComponent(name)}/stop`);
      return contracts.vmResponseSchema.parse(data);
    },
    deleteVm: async (name: string) => {
      const { data } = await apiClient.delete(`/vms/${encodeURIComponent(name)}`);
      return contracts.vmDeletionSchema.parse(data);
    },
    createConsoleTicket: async (name: string, options: { readonly signal?: AbortSignal } = {}) => {
      const { data } = await apiClient.post(`/vms/${encodeURIComponent(name)}/console-ticket`, undefined, options);
      return contracts.consoleTicketSchema.parse(data);
    },
    getOffers: async () => {
      const { data } = await apiClient.get('/vm-offers');
      return contracts.offersSchema.parse(data);
    },
    claimOffer: async (name: string, payload: ClaimVmOfferPayload) => {
      const { data } = await apiClient.post(`/vm-offers/${encodeURIComponent(name)}/claim`, payload);
      return contracts.vmResponseSchema.parse(data);
    },
  };
  const adminApi = {
    getUsers: async () => {
      const { data } = await apiClient.get('/admin/users');
      return contracts.usersSchema.parse(data);
    },
    updateUserAccess: async (nameOrUsername: string, accessLevel: AccessLevel) => {
      const { data } = await apiClient.patch(`/admin/users/${encodeURIComponent(nameOrUsername)}/access-level`, { access_level: accessLevel });
      return contracts.meSchema.parse(data);
    },
    deleteUser: async (nameOrUsername: string) => {
      const { data } = await apiClient.delete(`/admin/users/${encodeURIComponent(nameOrUsername)}`);
      return contracts.messageSchema.parse(data);
    },
    getVms: async () => {
      const { data } = await apiClient.get('/admin/vms');
      return contracts.globalVmsSchema.parse(data);
    },
    forceStopVm: async (namespace: string, name: string) => {
      const { data } = await apiClient.patch(`/admin/vms/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/power`, { powerState: 'Off' });
      return contracts.vmResponseSchema.parse(data);
    },
    deleteVm: async (namespace: string, name: string) => {
      const { data } = await apiClient.delete(`/admin/vms/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}`);
      return contracts.vmDeletionSchema.parse(data);
    },
    getSettings: async () => {
      const { data } = await apiClient.get('/admin/settings');
      return settings.adminConfigSchema.parse(data);
    },
    saveDomain: async (baseDomain: string) => {
      const { data } = await apiClient.post('/admin/domain', { baseDomain });
      return settings.settingsMutationSchema.parse(data);
    },
    saveHTTPSPolicy: async (payload: HTTPSPolicyPayload) => {
      const { data } = await apiClient.post('/admin/https', payload);
      return settings.settingsMutationSchema.parse(data);
    },
    saveSSHGateway: async (payload: SSHGatewayPayload) => {
      const { data } = await apiClient.post('/admin/ssh-gateway', payload);
      return settings.settingsMutationSchema.parse(data);
    },
    saveAdminContact: async (adminContact: string) => {
      const { data } = await apiClient.post('/admin/admin-contact', { adminContact });
      return settings.settingsMutationSchema.parse(data);
    },
    createVmOffer: async (payload: AdminCreateVmOfferPayload) => {
      const { data } = await apiClient.post('/admin/vm-offers', payload);
      return contracts.offerResponseSchema.parse(data);
    },
    deleteVmOffer: async (namespace: string, name: string) => {
      const { data } = await apiClient.delete(`/admin/vm-offers/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}`);
      return contracts.messageSchema.parse(data);
    },
    rotateRuntimeSecrets: async (payload: RuntimeSecretRotation) => {
      const { data } = await apiClient.post('/admin/runtime-secrets/rotate', payload);
      return settings.settingsMutationSchema.parse(data);
    },
    saveCert: async (payload: CertPayload) => {
      const { data } = await apiClient.post('/admin/cert', payload);
      return settings.settingsMutationSchema.parse(data);
    },
  };
  return { authApi, configApi, vmApi, adminApi };
}
