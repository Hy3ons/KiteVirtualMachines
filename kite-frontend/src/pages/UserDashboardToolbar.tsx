import React from 'react';
import { Button, Space, Tooltip, Typography } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';

const { Title } = Typography;

type UserDashboardToolbarProps = {
  readonly canCreateVm: boolean;
  readonly loading: boolean;
  readonly accessLevel: number;
  readonly adminContact: string;
  readonly onRefresh: () => void;
  readonly onCreate: () => void;
};

export const UserDashboardToolbar: React.FC<UserDashboardToolbarProps> = ({
  canCreateVm,
  loading,
  accessLevel,
  adminContact,
  onRefresh,
  onCreate,
}) => {
  const disabledReason = accessLevel < 1
    ? `VM 생성 권한이 없습니다.${adminContact ? ` 관리자 연락처: ${adminContact}` : ''}`
    : 'Level 1 VM 생성 한도에 도달했습니다.';

  return (
    <div className="dashboard-toolbar">
      <Title level={2} style={{ margin: 0 }}>My Virtual Machines</Title>
      <Space wrap>
        <Tooltip title="목록 새로고침">
          <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading} />
        </Tooltip>
        <Tooltip title={!canCreateVm ? disabledReason : ''}>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            size="large"
            disabled={!canCreateVm}
            onClick={onCreate}
          >
            Create VM
          </Button>
        </Tooltip>
      </Space>
    </div>
  );
};
