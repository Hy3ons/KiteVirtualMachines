import { z } from 'zod';

export const gatewayPhaseSchema = z.enum(['Disabled', 'Reconciling', 'Ready', 'Blocked', 'Failed']);
export const gatewayStatusSchema = z.object({
  phase: gatewayPhaseSchema,
  reason: z.string(),
  message: z.string(),
  observedExternalPort: z.string().optional(),
  observedServiceName: z.string().optional(),
  lastTransitionTime: z.string().optional(),
});
const gatewayFields = {
  externalEnabled: z.boolean(),
  externalPort: z.string(),
  publicPort: z.string(),
};
export const gatewaySettingsSchema = z.object({ ...gatewayFields, status: gatewayStatusSchema });

// Go's public settings flatten status; consumers use the same nested shape as admin settings.
const publicGatewaySchema = z.object({
  ...gatewayFields,
  phase: gatewayPhaseSchema,
  reason: z.string(),
  message: z.string(),
}).transform(({ phase, reason, message, ...settings }) => ({
  ...settings,
  status: { phase, reason, message },
}));
const configFields = {
  baseDomain: z.string(),
  forceHttps: z.boolean(),
  adminContact: z.string(),
  hasTLSCertificate: z.boolean(),
};
export const publicConfigSchema = z.object({
  config: z.object({ ...configFields, sshGateway: publicGatewaySchema }),
});
export const adminConfigSchema = z.object({
  config: z.object({
    ...configFields,
    hasJWTSecret: z.boolean(),
    hasPasswordSalt: z.boolean(),
    sshGateway: gatewaySettingsSchema,
  }),
});
export const settingsMutationSchema = adminConfigSchema.extend({ message: z.string() });
