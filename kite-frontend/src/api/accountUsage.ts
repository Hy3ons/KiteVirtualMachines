import { vmsSchema } from './contracts';
import type { UserVm } from './types';

export type AccountVmUsage = { readonly total: number; readonly running: number };

// parseAccountVmUsage validates the existing personal /vms response for the account popover.
// input is endpoint JSON; stopped/creating VMs count as held resources, but deleting VMs do not.
export function parseAccountVmUsage(input: unknown): AccountVmUsage {
  return accountVmUsage(vmsSchema.parse(input).vms);
}

// accountVmUsage aggregates resources that already crossed the endpoint boundary.
export function accountVmUsage(vms: readonly UserVm[]): AccountVmUsage {
  const held = vms.filter((vm) => !vm.delete);
  return { total: held.length, running: held.filter((vm) => vm.phase === 'Running').length };
}
