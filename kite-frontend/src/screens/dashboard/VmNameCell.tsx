import { Popover } from 'antd';
import type { DashboardVm } from '../UserDashboardTypes';

type VmNameCellProps = {
  readonly vm: DashboardVm;
  readonly onNavigate: (name: string) => void;
};

// VmNameCell exposes API-backed VM specs on hover or keyboard focus.
// vm supplies the row identity and specs; onNavigate opens its existing detail route.
export function VmNameCell({ vm, onNavigate }: VmNameCellProps) {
  return (
    <Popover title={vm.name} placement="topLeft" trigger={['hover', 'focus']} mouseEnterDelay={0.15} content={
      <dl className="workspace-vm-specs">
        <div><dt>CPU</dt><dd>{vm.cpu}<small>vCPU</small></dd></div>
        <div><dt>메모리</dt><dd>{vm.memory}</dd></div>
        <div><dt>디스크</dt><dd>{vm.disk}</dd></div>
      </dl>
    }>
      <a href={`/dashboard/kite-machine/${encodeURIComponent(vm.name)}`} className="workspace-vm-name" onClick={(event) => {
        event.preventDefault();
        onNavigate(vm.name);
      }}>{vm.name}</a>
    </Popover>
  );
}
