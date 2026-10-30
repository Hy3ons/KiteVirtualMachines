import type { UserVm } from '../api/types';

// Shared observed-state rules keep row, detail, and console controls consistent.
export function vmLifecycle(vm: Pick<UserVm, 'phase' | 'delete'>) {
  const running = vm.phase === 'Running';
  return {
    running,
    canConnect: running && !vm.delete,
    canMutate: !vm.delete,
    nextPower: running ? 'Off' : 'On',
  } as const;
}
