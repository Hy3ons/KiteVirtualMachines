export const user = { username: 'demo', email: 'demo@example.test', namespace: 'demo-ns', access_level: 1, profile_image: '' };
export const vm = { id: 'ns/demo', name: 'demo', namespace: 'ns', owner: 'ns', domain: '', phase: 'Running', powerState: 'On', currentPowerState: 'On', cpu: 2, memory: '4Gi', image: 'ubuntu', disk: '30Gi', domainPrefix: '', sshId: 'ubuntu', delete: false, dataVolumePhase: 'Succeeded', dataVolumeProgress: '100%', dataVolumeMessage: '' };
export const offer = { id: 'ns/offer', name: 'offer', namespace: 'ns', cpu: 2, memory: '4Gi', disk: '30Gi', image: 'ubuntu', expiresAt: '2030-01-01T00:00:00Z', createdBy: 'admin', phase: 'Available', claimedBy: '', message: '' };
export const gatewayStatus = { phase: 'Disabled', reason: 'ExternalDisabled', message: '' };
export const publicConfig = { baseDomain: 'example.test', forceHttps: false, adminContact: '', hasTLSCertificate: true, sshGateway: { externalEnabled: false, externalPort: '22', publicPort: '22', ...gatewayStatus } };
export const adminConfig = { ...publicConfig, hasJWTSecret: true, hasPasswordSalt: true, sshGateway: { externalEnabled: false, externalPort: '22', publicPort: '22', status: gatewayStatus } };
