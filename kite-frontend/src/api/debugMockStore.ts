import { debugAdminApi } from './debugMockAdmin';
import { debugAuthApi } from './debugMockAuth';
import { debugOfferApi } from './debugMockOffer';
import { debugVmApi } from './debugMockVm';

export const debugMockApi = {
  ...debugAuthApi,
  ...debugVmApi,
  ...debugOfferApi,
  ...debugAdminApi,
  getConfig: debugAdminApi.getSettings,
};
