import React from 'react';
import { Button, Tooltip } from 'antd';
import { DesktopOutlined } from '@ant-design/icons';
import type { DashboardVm } from './UserDashboardTypes';

type VmConsoleButtonProps = {
  readonly vm: DashboardVm;
  readonly onOpen: (name: string) => void;
};

export const VmConsoleButton: React.FC<VmConsoleButtonProps> = ({ vm, onOpen }) => (
  <Tooltip title={vm.phase === 'Running' ? 'Open serial console' : 'VM이 Running 상태일 때 사용할 수 있습니다.'}>
    <Button
      type="text"
      icon={<DesktopOutlined />}
      disabled={vm.phase !== 'Running'}
      onClick={() => onOpen(vm.name)}
    >
      Console
    </Button>
  </Tooltip>
);
