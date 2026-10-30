import type { AccessLevel, CreateVmPayload } from '../api/types';

export const LEVEL_1_FIXED_CPU = 2;
export const LEVEL_1_FIXED_MEMORY = '4Gi';
export const LEVEL_1_FIXED_DISK_GI = 25;
export const MIN_DISK_GI = 20;
export const LEVEL_1_VM_QUOTA = 2;

// Access predicates fail closed for an unavailable session throughout the workspace.
export const canControlVm = (level: AccessLevel | null): boolean => level !== null;
export const canManageUsers = (level: AccessLevel | null): boolean => (level ?? 0) >= 2;
export const canManageSystem = (level: AccessLevel | null): boolean => level === 3;

// Counts include deleting VMs until the API removes them from the inventory.
export function creationPolicy(level: AccessLevel | null, inventoryCount: number): 'allowed' | 'forbidden' | 'quota' {
  if (level === null || level === 0) return 'forbidden';
  if (level === 1 && inventoryCount >= LEVEL_1_VM_QUOTA) return 'quota';
  return 'allowed';
}

// Applies role resource limits to form input before sending a creation request.
export function buildVmCreatePayload(level: AccessLevel, values: Omit<CreateVmPayload, 'powerState' | 'disk'> & { readonly disk: number }): CreateVmPayload {
  const resources = level === 1
    ? { cpu: LEVEL_1_FIXED_CPU, memory: LEVEL_1_FIXED_MEMORY, disk: LEVEL_1_FIXED_DISK_GI }
    : { disk: Math.max(MIN_DISK_GI, values.disk), ...(values.cpu === undefined ? {} : { cpu: values.cpu }), ...(values.memory === undefined ? {} : { memory: values.memory }) };
  return { ...values, ...resources, powerState: 'On' };
}
