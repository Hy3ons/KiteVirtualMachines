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

export type KiteUser = {
  readonly access_level: number;
  readonly username: string;
  readonly namespace: string;
  readonly profile_image: string;
};

export type AuthResponse = {
  readonly expiresIn: number;
  readonly expiresAt: string;
  readonly user: KiteUser;
};

export type LogoutResponse = {
  readonly message: string;
};

export type SignupResponse = {
  readonly message: string;
  readonly user: KiteUser & {
    readonly email: string;
  };
};

export type ConfigResponse = {
  readonly config: {
    readonly baseDomain: string;
    readonly forceHttps: boolean;
    readonly hasJWTSecret: boolean;
    readonly hasPasswordSalt: boolean;
    readonly hasTLSCertificate: boolean;
  };
};

export type VmPhase = 'Running' | 'Stopped' | 'Creating' | 'Terminating';

export type UserVm = {
  readonly id: string;
  readonly name: string;
  readonly namespace: string;
  readonly owner: string;
  readonly domain: string;
  readonly phase: VmPhase;
  readonly powerState: 'On' | 'Off';
  readonly currentPowerState: 'On' | 'Off' | '';
  readonly cpu: number;
  readonly memory: string;
  readonly image: string;
  readonly disk: string;
  readonly domainPrefix: string;
  readonly sshId: string;
  readonly delete: boolean;
  readonly dataVolumePhase: string;
  readonly dataVolumeProgress: string;
  readonly dataVolumeMessage: string;
};

export type VmsResponse = {
  readonly vms: readonly UserVm[];
};

export type VmResponse = {
  readonly vm: UserVm;
};

export type ConsoleTicketResponse = {
  readonly ticket: string;
  readonly expiresAt: string;
};

export type CreateVmPayload = {
  readonly name: string;
  readonly domainPrefix?: string;
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

export type AdminUser = {
  readonly username: string;
  readonly email: string;
  readonly namespace: string;
  readonly accessLevel: number;
  readonly status: 'Active' | 'Pending';
};

export type AdminUsersResponse = {
  readonly users: readonly AdminUser[];
};

export type GlobalVm = {
  readonly id: string;
  readonly owner: string;
  readonly namespace: string;
  readonly name: string;
  readonly domain: string;
  readonly phase: VmPhase;
  readonly cpu: number;
  readonly memory: string;
};

export type GlobalVmsResponse = {
  readonly vms: readonly GlobalVm[];
};

export type RuntimeSecretRotation = {
  readonly rotateJWTSecret?: boolean;
  readonly rotatePasswordSalt?: boolean;
};

export type HTTPSPolicyPayload = {
  readonly forceHttps: boolean;
};

export type CertPayload = {
  readonly tlsCert: string;
  readonly tlsKey: string;
};
