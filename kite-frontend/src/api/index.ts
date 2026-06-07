import { apiClient } from './axios';

const useMockApi = import.meta.env.VITE_USE_MOCK === 'true';

// ========================
// Auth
// ========================
export const authApi = {
  login: async (credentials: any) => {
    // 로컬 DEV 환경에서 백엔드 없이 UI 퍼블리싱/수정을 위한 admin 우회 로직
    if (useMockApi && credentials.email === 'admin') {
      return {
        accessToken: 'dummy-admin-token',
        user: { access_level: 3, username: 'admin', namespace: 'system', profile_image: '' }
      };
    }
    const { data } = await apiClient.post('/auth/login', credentials);
    return data;
  },
  signup: async (payload: any) => {
    const { data } = await apiClient.post('/auth/signup', payload);
    return data;
  },
  getMe: async () => {
    const { data } = await apiClient.get('/me');
    return data;
  },
};

// ========================
// Config
// ========================
export const configApi = {
  getConfig: async () => {
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
      return { vms: [
        { id: '1', name: 'dev-vm-1', domain: 'dev.apps.example.com', phase: 'Running', cpu: 2, memory: '4Gi', disk: '30Gi', sshId: 'ubuntu' },
        { id: '2', name: 'test-db', domain: 'db.apps.example.com', phase: 'Stopped', cpu: 2, memory: '4Gi', disk: '50Gi', sshId: 'dbuser' },
      ]};
    }
    const { data } = await apiClient.get('/vms');
    return data;
  },
  getVm: async (name: string) => {
    const { data } = await apiClient.get(`/vms/${name}`);
    return data;
  },
  createVm: async (payload: any) => {
    const { data } = await apiClient.post('/vms', payload);
    return data;
  },
  updateVm: async (name: string, payload: any) => {
    const { data } = await apiClient.patch(`/vms/${name}`, payload);
    return data;
  },
  startVm: async (name: string) => {
    const { data } = await apiClient.post(`/vms/${name}/start`);
    return data;
  },
  stopVm: async (name: string) => {
    const { data } = await apiClient.post(`/vms/${name}/stop`);
    return data;
  },
  deleteVm: async (name: string) => {
    const { data } = await apiClient.delete(`/vms/${name}`);
    return data;
  },
};

// ========================
// Admin
// ========================
export const adminApi = {
  getUsers: async () => {
    if (useMockApi) {
      return { users: [
        { username: 'admin', email: 'admin@kite.com', namespace: 'system', accessLevel: 3, status: 'Active' },
        { username: 'hyeonseok', email: 'hyeonseok@kite.com', namespace: 'hyeonseok-ns', accessLevel: 1, status: 'Active' }
      ]};
    }
    const { data } = await apiClient.get('/admin/users');
    if (data && data.users) {
      data.users = data.users.map((u: any) => ({
        ...u,
        accessLevel: u.access_level
      }));
    }
    return data;
  },
  updateUserAccess: async (nameOrUsername: string, accessLevel: number) => {
    const { data } = await apiClient.patch(`/admin/users/${nameOrUsername}/access-level`, { access_level: accessLevel });
    return data;
  },
  deleteUser: async (nameOrUsername: string) => {
    const { data } = await apiClient.delete(`/admin/users/${nameOrUsername}`);
    return data;
  },
  getVms: async () => {
    if (useMockApi) {
      return { vms: [
        { id: '1', owner: 'hyeonseok-ns', namespace: 'hyeonseok-ns', name: 'dev-vm-1', domain: 'dev.apps.example.com', phase: 'Running', cpu: 2, memory: '4Gi' }
      ]};
    }
    const { data } = await apiClient.get('/admin/vms');
    return data;
  },
  forceStopVm: async (namespace: string, name: string) => {
    const { data } = await apiClient.patch(`/admin/vms/${namespace}/${name}/power`, { powerState: 'Off' });
    return data;
  },
  deleteVm: async (namespace: string, name: string) => {
    const { data } = await apiClient.delete(`/admin/vms/${namespace}/${name}`);
    return data;
  },
  getSettings: async () => {
    const { data } = await apiClient.get('/admin/settings');
    return data;
  },
  saveDomain: async (baseDomain: string) => {
    const { data } = await apiClient.post('/admin/domain', { baseDomain });
    return data;
  },
  rotateRuntimeSecrets: async (payload: { rotateJWTSecret?: boolean; rotatePasswordSalt?: boolean }) => {
    const { data } = await apiClient.post('/admin/runtime-secrets/rotate', payload);
    return data;
  },
  saveCert: async (payload: any) => {
    const { data } = await apiClient.post('/admin/cert', payload);
    return data;
  },
};
