import React from 'react';
import { Button, Tooltip } from 'antd';
import { DesktopOutlined } from '@ant-design/icons';
import type { DashboardVm } from './UserDashboardTypes';

type VmConsoleButtonProps = {
  readonly vm: DashboardVm;
  readonly onOpen: (name: string) => void;
};

// VmConsoleButton opens the selected VM console only when its observed phase is Running.
export const VmConsoleButton: React.FC<VmConsoleButtonProps> = ({ vm, onOpen }) => (
  <Tooltip title={vm.phase === 'Running' ? '시리얼 콘솔 열기' : 'VM이 Running 상태일 때 사용할 수 있습니다.'}>
    <Button
      type="text"
      aria-label={`${vm.name} 웹 콘솔 열기`}
      icon={<DesktopOutlined />}
      disabled={vm.phase !== 'Running'}
      onClick={() => onOpen(vm.name)}
    >
      콘솔
    </Button>
  </Tooltip>
);
