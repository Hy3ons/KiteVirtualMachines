import { StatusBadge } from '../components/StatusBadge';
import { VmNameCell } from './dashboard/VmNameCell';
import { VmRowActions } from './dashboard/VmRowActions';
import type { TableColumnsType } from 'antd';
import type { DashboardVm } from './UserDashboardTypes';

type UserDashboardColumnsOptions = {
  readonly onNavigateToDetail: (name: string) => void;
  readonly onConnect: (vm: DashboardVm) => void;
  readonly onOpenConsole: (name: string) => void;
  readonly onStart: (name: string) => Promise<void>;
  readonly onStop: (name: string) => Promise<void>;
  readonly onDelete: (name: string) => Promise<void>;
  readonly canMutateVm: boolean;
};

// createUserDashboardColumns builds a compact VM list with hover specs and a row action menu.
// options binds the existing navigation and Go-backed operations without changing their payloads.
export const createUserDashboardColumns = (options: UserDashboardColumnsOptions): TableColumnsType<DashboardVm> => [
  {
    title: '가상머신 이름', dataIndex: 'name', key: 'name', width: 200,
    ellipsis: { showTitle: false },
    render: (_: string, vm) => <VmNameCell vm={vm} onNavigate={options.onNavigateToDetail} />,
  },
  { title: 'Domain', dataIndex: 'domain', key: 'domain', width: 260, ellipsis: true },
  { title: 'SSH ID', dataIndex: 'sshId', key: 'sshId', width: 140, ellipsis: true },
  { title: '상태', dataIndex: 'phase', key: 'phase', width: 120, render: (phase: string) => <StatusBadge phase={phase} /> },
  {
    title: '작업', key: 'actions', width: 80, align: 'center',
    render: (_, vm) => <VmRowActions vm={vm} canMutate={options.canMutateVm} onConnect={options.onConnect} onOpenConsole={options.onOpenConsole} onStart={options.onStart} onStop={options.onStop} onDelete={options.onDelete} />,
  },
];
