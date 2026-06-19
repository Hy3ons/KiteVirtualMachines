import type { FormInstance } from 'antd';
import type { CreateVmPayload, UserVm } from '../api/types';

export type DashboardVm = UserVm;

export type VmCreateFormValues = {
  readonly name: string;
  readonly domainPrefix?: string;
  readonly sshId: string;
  readonly sshPassword: string;
  readonly disk: number;
  readonly cpu?: number;
  readonly memory?: string;
};

export type VmCreatePayload = CreateVmPayload;

export type VmCreateForm = FormInstance<VmCreateFormValues>;
