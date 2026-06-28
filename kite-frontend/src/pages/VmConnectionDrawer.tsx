import React from 'react';
import { Alert, Drawer, Typography } from 'antd';
import { MOCK_ENV } from '../config/mockEnv';
import type { DashboardVm } from './UserDashboardTypes';

const { Title, Text, Paragraph } = Typography;

type VmConnectionDrawerProps = {
  readonly open: boolean;
  readonly vm: DashboardVm | null;
  readonly onClose: () => void;
};

export const VmConnectionDrawer: React.FC<VmConnectionDrawerProps> = ({ open, vm, onClose }) => (
  <Drawer
    title={<Text strong className="drawer-title">SSH 접속 가이드: {vm?.name}</Text>}
    placement="right"
    size={600}
    onClose={onClose}
    open={open}
  >
    {vm && (
      <div className="drawer-stack">
        <Alert
          title="SSH 접속 안내"
          description="VM 생성 시 입력한 SSH User ID와 SSH Password로 서버에 접속하면 kite-gateway가 해당 VM SSH 서비스로 연결합니다."
          type="info"
          showIcon
        />

        <div>
          <Title level={5}>방법 1. 한 줄 명령어로 바로 접속하기</Title>
          <Paragraph>Kite가 설치된 서버 주소로 접속합니다.</Paragraph>
          <pre className="code-block">
            <code>ssh {vm.sshId}@{MOCK_ENV.BASE_DOMAIN}</code>
          </pre>
        </div>

        <div>
          <Title level={5}>방법 2. ~/.ssh/config 수정하고 편하게 들어가기</Title>
          <Paragraph>아래 내용을 <Text keyboard>~/.ssh/config</Text>에 추가하세요.</Paragraph>
          <pre className="code-block">
            <code>{`Host bastion
    HostName ${MOCK_ENV.BASE_DOMAIN}
    User ${vm.sshId}
    Port 22
`}</code>
          </pre>
          <Paragraph>이후에는 터미널에서 <Text keyboard>ssh bastion</Text> 만 입력하면 해당 VM으로 접속됩니다.</Paragraph>
        </div>
      </div>
    )}
  </Drawer>
);
