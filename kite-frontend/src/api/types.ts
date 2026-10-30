import type { z } from 'zod';
import type {
  sessionUserSchema,
  authResponseSchema,
  messageSchema,
  signupResponseSchema,
  vmPhaseSchema,
  vmSchema,
  vmsSchema,
  vmResponseSchema,
  offerPhaseSchema,
  offerSchema,
  offersSchema,
  offerResponseSchema,
  consoleTicketSchema,
  adminUserSchema,
  usersSchema,
  globalVmSchema,
  globalVmsSchema,
} from './contracts';
import type {
  gatewayPhaseSchema,
  gatewayStatusSchema,
  gatewaySettingsSchema,
  adminConfigSchema,
  publicConfigSchema,
} from './configContracts';
export type PublicConfigResponse = Readonly<z.infer<typeof publicConfigSchema>>;
export type AdminConfigResponse = ConfigResponse;
export type AccessLevel = KiteUser['access_level'];
export type LoginCredentials = {
  readonly email: string;
  readonly password: string;
};

export type SignupPayload = {
  readonly username: string;
  readonly email: string;
  readonly password: string;
  readonly profile_image: string;
};

export type KiteUser = Readonly<z.infer<typeof sessionUserSchema>>;

export type AuthResponse = Readonly<z.infer<typeof authResponseSchema>>;

export type LogoutResponse = Readonly<z.infer<typeof messageSchema>>;

export type SignupResponse = Readonly<z.infer<typeof signupResponseSchema>>;

export type SSHGatewayPhase = z.infer<typeof gatewayPhaseSchema>;

export type SSHGatewayStatus = Readonly<z.infer<typeof gatewayStatusSchema>>;

export type SSHGatewaySettings = Readonly<z.infer<typeof gatewaySettingsSchema>>;

export type ConfigResponse = Readonly<z.infer<typeof adminConfigSchema>>;

export type VmPhase = z.infer<typeof vmPhaseSchema>;

export type UserVm = Readonly<z.infer<typeof vmSchema>>;

export type VmsResponse = Readonly<z.infer<typeof vmsSchema>>;

export type VmResponse = Readonly<z.infer<typeof vmResponseSchema>>;

export type VmOfferPhase = z.infer<typeof offerPhaseSchema>;

export type VmOffer = Readonly<z.infer<typeof offerSchema>>;

export type VmOffersResponse = Readonly<z.infer<typeof offersSchema>>;

export type ClaimVmOfferPayload = {
  readonly vmName: string;
  readonly domainPrefix: string;
  readonly sshId: string;
  readonly initialLoginPassword: string;
  readonly powerState?: 'On' | 'Off';
};

export type AdminCreateVmOfferPayload = {
  readonly targetUser?: string;
  readonly targetNamespace?: string;
  readonly cpu: number;
  readonly memory: string;
  readonly disk: string;
  readonly image?: string;
  readonly expiresAt?: string;
};

export type VmOfferResponse = Readonly<z.infer<typeof offerResponseSchema>>;

export type ConsoleTicketResponse = Readonly<z.infer<typeof consoleTicketSchema>>;

export type CreateVmPayload = {
  readonly name: string;
  readonly domainPrefix: string;
  readonly sshId: string;
  readonly sshPassword: string;
  readonly cpu?: number;
  readonly memory?: string;
  readonly disk: number | string;
  readonly powerState?: 'On' | 'Off';
};

export type UpdateVmPayload = {
  readonly domainPrefix?: string;
  readonly powerState?: 'On' | 'Off';
};

export type AdminUser = Readonly<z.infer<typeof adminUserSchema>>;

export type AdminUsersResponse = Readonly<z.infer<typeof usersSchema>>;

export type GlobalVm = Readonly<z.infer<typeof globalVmSchema>>;

export type GlobalVmsResponse = Readonly<z.infer<typeof globalVmsSchema>>;

export type RuntimeSecretRotation = {
  readonly rotateJWTSecret?: boolean;
  readonly rotatePasswordSalt?: boolean;
};

export type HTTPSPolicyPayload = {
  readonly forceHttps: boolean;
};

export type SSHGatewayPayload = {
  readonly externalEnabled: boolean;
  readonly externalPort: string;
  readonly publicPort: string;
};

export type CertPayload = {
  readonly tlsCert: string;
  readonly tlsKey: string;
};
