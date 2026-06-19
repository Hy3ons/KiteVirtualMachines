import { debugAdminApi } from './debugMockAdmin';
import { debugAuthApi } from './debugMockAuth';
import { debugVmApi } from './debugMockVm';

export const debugMockApi = {
  ...debugAuthApi,
  ...debugVmApi,
  ...debugAdminApi,
  getConfig: debugAdminApi.getSettings,
};
