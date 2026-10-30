import type { DashboardVm } from '../../../screens/UserDashboardTypes';
import type { SessionUser } from '../../../api/session';

export const DEMO_USER: SessionUser = { username: 'builder', email: 'builder@kite.local', namespace: 'demo', access_level: 1, profile_image: '', sessionExpiresAt: null };
export const DEMO_VM: DashboardVm = {
  id: 'demo-vm', name: 'demo-vm', namespace: 'demo', owner: 'builder',
  domain: 'demo-vm.apps.example.com', domainPrefix: 'demo-vm', sshId: 'builder',
  phase: 'Running', powerState: 'On', currentPowerState: 'On', cpu: 2, memory: '4Gi', disk: '25Gi',
  image: 'ubuntu-22.04', delete: false, dataVolumePhase: 'Succeeded', dataVolumeProgress: '100%', dataVolumeMessage: '',
};

// Demo callbacks intentionally affect no resources; the inert canvas is driven only by story time.
export const demoIgnore = () => undefined;
export const demoIgnoreAsync = async () => undefined;
