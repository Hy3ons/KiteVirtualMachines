import { z } from 'zod';

const usageResponseSchema = z.object({ vms: z.array(z.object({ delete: z.boolean(), phase: z.string() })) });
export type AccountVmUsage = { readonly total: number; readonly running: number };

// parseAccountVmUsage validates the existing personal /vms response for the account popover.
// input is endpoint JSON; stopped/creating VMs count as held resources, but deleting VMs do not.
export function parseAccountVmUsage(input: unknown): AccountVmUsage {
  const { vms } = usageResponseSchema.parse(input);
  const held = vms.filter((vm) => !vm.delete);
  return { total: held.length, running: held.filter((vm) => vm.phase === 'Running').length };
}
