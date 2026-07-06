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

export type SSHGatewayPhase = 'Disabled' | 'Reconciling' | 'Ready' | 'Blocked' | 'Failed';

export type SSHGatewayStatus = {
  readonly phase: SSHGatewayPhase;
  readonly reason: string;
  readonly message: string;
  readonly observedExternalPort?: string;
  readonly observedHostFallbackAddress?: string;
  readonly observedServiceName?: string;
  readonly lastTransitionTime?: string;
};

export type SSHGatewaySettings = {
  readonly externalEnabled: boolean;
  readonly externalPort: string;
  readonly hostFallbackEnabled?: boolean;
  readonly hostSshdPort?: string;
  readonly phase?: SSHGatewayPhase;
  readonly reason?: string;
  readonly message?: string;
  readonly status?: SSHGatewayStatus;
};

export type ConfigResponse = {
  readonly config: {
    readonly baseDomain: string;
    readonly forceHttps: boolean;
    readonly adminContact: string;
    readonly hasJWTSecret?: boolean;
    readonly hasPasswordSalt?: boolean;
    readonly hasTLSCertificate?: boolean;
    readonly sshGateway: SSHGatewaySettings;
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

export type VmOfferPhase = 'Available' | 'Claimed' | 'Expired';

export type VmOffer = {
  readonly id: string;
  readonly name: string;
  readonly namespace: string;
  readonly cpu: number;
  readonly memory: string;
  readonly disk: string;
  readonly image: string;
  readonly expiresAt: string;
  readonly createdBy: string;
  readonly phase: VmOfferPhase;
  readonly claimedBy: string;
  readonly message: string;
};

export type VmOffersResponse = {
  readonly offers: readonly VmOffer[];
};

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

export type VmOfferResponse = {
  readonly offer: VmOffer;
};

export type ConsoleTicketResponse = {
  readonly ticket: string;
  readonly expiresAt: string;
};

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

export type SSHGatewayPayload = {
  readonly externalEnabled: boolean;
  readonly externalPort: string;
  readonly hostFallbackEnabled: boolean;
  readonly hostSshdPort: string;
};

export type CertPayload = {
  readonly tlsCert: string;
  readonly tlsKey: string;
};
