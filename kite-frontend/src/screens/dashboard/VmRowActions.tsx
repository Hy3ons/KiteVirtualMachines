import { vmLifecycle } from '../../domain/vmLifecycle';
import { useState } from 'react';
import { App, Button, Dropdown } from 'antd';
import { MoreOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';
import type { DashboardVm } from '../UserDashboardTypes';

type VmRowActionsProps = {
  readonly vm: DashboardVm;
  readonly canMutate: boolean;
  readonly onConnect: (vm: DashboardVm) => void;
  readonly onOpenConsole: (name: string) => void;
  readonly onStart: (name: string) => Promise<void>;
  readonly onStop: (name: string) => Promise<void>;
  readonly onDelete: (name: string) => Promise<void>;
  readonly previewOpen?: boolean;
};

// VmRowActions groups the existing row callbacks in a text-only dropdown.
// vm and canMutate determine availability; deletion still requires explicit confirmation.
export function VmRowActions({ vm, canMutate, onConnect, onOpenConsole, onStart, onStop, onDelete, previewOpen }: VmRowActionsProps) {
  const { modal } = App.useApp();
  const [localOpen, setOpen] = useState(false);
  const open = previewOpen ?? localOpen;
  const { running, canConnect, canMutate: lifecycleCanMutate } = vmLifecycle(vm);
  const items: MenuProps['items'] = [
    { key: 'ssh', label: 'SSH 연결', disabled: !canConnect, onClick: () => onConnect(vm) },
    { key: 'console', label: '콘솔', disabled: !canConnect, onClick: () => onOpenConsole(vm.name) },
    { type: 'divider' },
    { key: 'power', label: running ? '중지하기' : '시작하기', disabled: !canMutate || !lifecycleCanMutate, onClick: () => { void (running ? onStop(vm.name) : onStart(vm.name)); } },
    { key: 'delete', label: '삭제하기', danger: true, disabled: !canMutate || !lifecycleCanMutate, onClick: () => {
      modal.confirm({
        title: `${vm.name}을(를) 삭제하시겠습니까?`,
        content: '가상머신과 연결된 데이터가 영구적으로 삭제됩니다.',
        okText: '삭제', cancelText: '취소', okButtonProps: { danger: true },
        autoFocusButton: 'cancel', onOk: () => onDelete(vm.name),
      });
    } },
  ];
  return (
    <Dropdown menu={{ items, onClick: () => setOpen(false) }} trigger={['click']} placement="bottomRight" open={open} onOpenChange={setOpen} autoFocus={previewOpen === undefined} classNames={{ root: 'workspace-vm-actions-menu' }}>
      <Button type="text" icon={<MoreOutlined />} aria-label={`${vm.name} 작업 메뉴`} aria-haspopup="menu" aria-expanded={open} />
    </Dropdown>
  );
}
