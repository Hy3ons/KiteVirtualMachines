import { z } from 'zod';

export const accessLevelSchema = z.union([z.literal(0), z.literal(1), z.literal(2), z.literal(3)]);
export const sessionUserSchema = z.object({
  username: z.string().min(1),
  email: z.string(),
  namespace: z.string().min(1),
  access_level: accessLevelSchema,
  profile_image: z.string().default(''),
});
export const meSchema = z.object({ user: sessionUserSchema });
export const sessionResponseSchema = meSchema.extend({ expiresAt: z.iso.datetime() });
export const messageSchema = z.object({ message: z.string() });
export const authResponseSchema = sessionResponseSchema.extend({ expiresIn: z.number().int().nonnegative() });
export const signupResponseSchema = meSchema.extend({ message: z.string() });
export const vmPhaseSchema = z.enum([
  'Running', 'Stopped', 'Creating', 'Terminating', 'Provisioning', 'Ready', 'Deleting', 'Failed',
]);
const cpuSchema = z.number().int().nonnegative();
export const vmSchema = z.object({
  id: z.string(),
  name: z.string(),
  namespace: z.string(),
  owner: z.string(),
  domain: z.string(),
  phase: vmPhaseSchema,
  powerState: z.enum(['On', 'Off']),
  currentPowerState: z.enum(['On', 'Off', '']),
  cpu: cpuSchema,
  memory: z.string(),
  image: z.string(),
  disk: z.string(),
  domainPrefix: z.string(),
  sshId: z.string(),
  delete: z.boolean(),
  dataVolumePhase: z.string(),
  dataVolumeProgress: z.string(),
  dataVolumeMessage: z.string(),
});
export const vmsSchema = z.object({ vms: z.array(vmSchema).readonly() });
export const vmResponseSchema = z.object({ vm: vmSchema });
export const vmDeletionSchema = vmResponseSchema.extend({ message: z.string() });
export const offerPhaseSchema = z.enum(['Available', 'Claimed', 'Expired']);
export const offerSchema = z.object({
  id: z.string(),
  name: z.string(),
  namespace: z.string(),
  cpu: cpuSchema,
  memory: z.string(),
  disk: z.string(),
  image: z.string(),
  expiresAt: z.string(),
  createdBy: z.string(),
  phase: offerPhaseSchema,
  claimedBy: z.string(),
  message: z.string(),
});
export const offersSchema = z.object({ offers: z.array(offerSchema).readonly() });
export const offerResponseSchema = z.object({ offer: offerSchema });
export const adminUserSchema = z.object({
  username: z.string(),
  email: z.string(),
  namespace: z.string(),
  accessLevel: accessLevelSchema,
  status: z.enum(['Active', 'Pending']),
});
const adminWireUserSchema = sessionUserSchema.extend({
  status: z.enum(['Active', 'Pending']).optional(),
}).transform(({ access_level, username, email, namespace, status }) => ({
  username,
  email,
  namespace,
  accessLevel: access_level,
  status: status ?? (access_level > 0 ? 'Active' as const : 'Pending' as const),
}));
export const usersSchema = z.object({ users: z.array(adminWireUserSchema).readonly() });
export const globalVmSchema = vmSchema.pick({
  id: true,
  owner: true,
  namespace: true,
  name: true,
  domain: true,
  phase: true,
  cpu: true,
  memory: true,
});
export const globalVmsSchema = z.object({ vms: z.array(globalVmSchema).readonly() });
export const consoleTicketSchema = z.object({ ticket: z.string(), expiresAt: z.string() });
