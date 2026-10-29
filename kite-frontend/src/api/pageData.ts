import { z } from 'zod';
import type { ApiClient } from './client';

const phase = z.enum(['Running', 'Stopped', 'Creating', 'Terminating', 'Provisioning', 'Ready', 'Deleting', 'Failed']);
const vmSchema = z.object({
  id: z.string(), name: z.string(), namespace: z.string(), owner: z.string(),
  domain: z.string(), phase, powerState: z.enum(['On', 'Off']),
  currentPowerState: z.enum(['On', 'Off', '']), cpu: z.number(), memory: z.string(),
  image: z.string(), disk: z.string(), domainPrefix: z.string(), sshId: z.string(),
  delete: z.boolean(), dataVolumePhase: z.string(), dataVolumeProgress: z.string(), dataVolumeMessage: z.string(),
});
const gatewayStatus = z.object({
  phase: z.enum(['Disabled', 'Reconciling', 'Ready', 'Blocked', 'Failed']),
  reason: z.string(), message: z.string(), observedExternalPort: z.string().optional(),
  observedServiceName: z.string().optional(), lastTransitionTime: z.string().optional(),
});
const configSchema = z.object({ config: z.object({
  baseDomain: z.string(), forceHttps: z.boolean(), adminContact: z.string(),
  hasJWTSecret: z.boolean().optional(), hasPasswordSalt: z.boolean().optional(), hasTLSCertificate: z.boolean().optional(),
  sshGateway: z.object({
    externalEnabled: z.boolean(), externalPort: z.string(), publicPort: z.string(),
    phase: gatewayStatus.shape.phase.optional(), reason: z.string().optional(), message: z.string().optional(), status: gatewayStatus.optional(),
  }),
}) });
const offersSchema = z.object({ offers: z.array(z.object({
  id: z.string(), name: z.string(), namespace: z.string(), cpu: z.number(),
  memory: z.string(), disk: z.string(), image: z.string(), expiresAt: z.string(),
  createdBy: z.string(), phase: z.enum(['Available', 'Claimed', 'Expired']), claimedBy: z.string(), message: z.string(),
})) });
export const usersSchema = z.object({ users: z.array(z.object({
  username: z.string(), email: z.string(), namespace: z.string(),
  access_level: z.number().int(), status: z.enum(['Active', 'Pending']).optional(),
}).transform(({ access_level, status, ...user }) => ({ ...user, accessLevel: access_level, status: status ?? (access_level > 0 ? 'Active' as const : 'Pending' as const) }))) });
const globalVmsSchema = z.object({ vms: z.array(z.object({
  id: z.string(), owner: z.string(), namespace: z.string(), name: z.string(),
  domain: z.string(), phase, cpu: z.number(), memory: z.string(),
})) });

// readData parses endpoint JSON with schema before passing it into SSR pages.
// client carries only this request's cookie; invalid API contracts fail visibly.
async function readData<T>(client: ApiClient, endpoint: string, schema: z.ZodType<T>): Promise<T> {
  const { data } = await client.get<unknown>(endpoint);
  return schema.parse(data);
}

// loadDashboard fetches independent VM, offer and connection settings in parallel.
// client is request-scoped; the result seeds the dashboard without a browser fetch waterfall.
export async function loadDashboard(client: ApiClient) {
  const [vms, offers, settings] = await Promise.all([
    readData(client, '/vms', z.object({ vms: z.array(vmSchema) })),
    readData(client, '/vm-offers', offersSchema),
    readData(client, '/config', configSchema),
  ]);
  return { vms: vms.vms, offers: offers.offers, config: settings.config };
}

// loadVm retrieves the named VM and connection configuration for SSR.
// name is encoded as one URL segment; client owns authentication and cache policy.
export async function loadVm(client: ApiClient, name: string) {
  const [result, settings] = await Promise.all([
    readData(client, `/vms/${encodeURIComponent(name)}`, z.object({ vm: vmSchema })),
    readData(client, '/config', configSchema),
  ]);
  return { vm: result.vm, config: settings.config };
}

// loadAdminDashboard loads users and, only for administrators, cluster-wide VMs.
// accessLevel comes from /me so manager sessions never request admin-only VM data.
export async function loadAdminDashboard(client: ApiClient, accessLevel: number) {
  const [users, vms] = await Promise.all([
    readData(client, '/admin/users', usersSchema),
    accessLevel >= 3 ? readData(client, '/admin/vms', globalVmsSchema) : Promise.resolve({ vms: [] }),
  ]);
  return { users: users.users, vms: vms.vms };
}

// loadSettings retrieves validated administrator settings for the SSR settings screen.
// client is created only after the page verifies access level 3.
export function loadSettings(client: ApiClient) {
  return readData(client, '/admin/settings', configSchema);
}

export type DashboardData = Awaited<ReturnType<typeof loadDashboard>>;
export type VmData = Awaited<ReturnType<typeof loadVm>>;
export type AdminDashboardData = Awaited<ReturnType<typeof loadAdminDashboard>>;
