import { readDebugState } from './debugMockState';
import { debugAdminApi } from './debugMockAdmin';
import { debugAuthApi } from './debugMockAuth';
import { debugOfferApi } from './debugMockOffer';
import { debugVmApi } from './debugMockVm';

export const debugMockApi = {
  ...debugAuthApi,
  ...debugVmApi,
  ...debugOfferApi,
  ...debugAdminApi,
  getConfig: async () => {
    const { baseDomain, forceHttps, adminContact, hasTLSCertificate, sshGateway } = readDebugState().config;
    const { status, ...gateway } = sshGateway;
    return { config: { baseDomain, forceHttps, adminContact, hasTLSCertificate, sshGateway: { ...gateway, phase: status.phase, reason: status.reason, message: status.message } } };
  },
};
