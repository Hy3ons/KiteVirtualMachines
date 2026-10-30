import type { ApiClient } from './client';
import type { AccessLevel } from './types';
import { createEndpoints } from './endpoints';
export { usersSchema } from './contracts';

// loadDashboard retrieves validated personal resources and public connection settings.
export async function loadDashboard(client: ApiClient) {
  const { vmApi, configApi } = createEndpoints(client);
  const [vms, offers, settings] = await Promise.all([vmApi.getVms(), vmApi.getOffers(), configApi.getConfig()]);
  return { vms: vms.vms, offers: offers.offers, config: settings.config };
}

// loadVm retrieves the encoded VM and public settings for its detail page.
export async function loadVm(client: ApiClient, name: string) {
  const { vmApi, configApi } = createEndpoints(client);
  const [result, settings] = await Promise.all([vmApi.getVm(name), configApi.getConfig()]);
  return { vm: result.vm, config: settings.config };
}

// loadConsoleVm retrieves only the validated VM needed by the console route.
export function loadConsoleVm(client: ApiClient, name: string) {
  return createEndpoints(client).vmApi.getVm(name);
}

// loadAdminDashboard avoids cluster-wide requests for managers below level three.
export async function loadAdminDashboard(client: ApiClient, accessLevel: AccessLevel) {
  const { adminApi } = createEndpoints(client);
  const [users, vms] = await Promise.all([adminApi.getUsers(), accessLevel >= 3 ? adminApi.getVms() : Promise.resolve({ vms: [] })]);
  return { users: users.users, vms: vms.vms };
}

// loadSettings retrieves the administrator-only settings contract.
export function loadSettings(client: ApiClient) {
  return createEndpoints(client).adminApi.getSettings();
}
export type DashboardData = Awaited<ReturnType<typeof loadDashboard>>;
export type VmData = Awaited<ReturnType<typeof loadVm>>;
export type AdminDashboardData = Awaited<ReturnType<typeof loadAdminDashboard>>;
