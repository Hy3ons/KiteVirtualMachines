import React from 'react';
import { Alert, Drawer, Typography } from 'antd';
import type { DashboardVm } from './UserDashboardTypes';

const { Title, Text, Paragraph } = Typography;

type VmConnectionDrawerProps = {
  readonly open: boolean;
  readonly vm: DashboardVm | null;
  readonly baseDomain: string;
  readonly onClose: () => void;
};

export const VmConnectionDrawer: React.FC<VmConnectionDrawerProps> = ({ open, vm, baseDomain, onClose }) => {
  const connectionHost = baseDomain.trim();
  const displayedHost = connectionHost || '<base-domain>';

  return (
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
            description="SSH 명령은 VM에 직접 비밀번호 로그인하지 않습니다. VM 생성 시 입력한 ID와 초기 비밀번호로 Kite gateway에 로그인하면, gateway가 Kite 관리 키로 해당 VM에 연결합니다."
            type="info"
            showIcon
          />

          {!connectionHost && (
            <Alert
              title="베이스 도메인 설정 필요"
              description="관리자 설정의 baseDomain을 불러오지 못했거나 아직 설정되지 않았습니다. 도메인 설정 후 다시 연결 안내를 확인하세요."
              type="warning"
              showIcon
            />
          )}

          <div>
            <Title level={5}>방법 1. 한 줄 명령어로 바로 접속하기</Title>
            <Paragraph>Kite gateway 도메인으로 접속합니다. 비밀번호 프롬프트에는 VM 생성 시 입력한 초기 비밀번호를 사용합니다.</Paragraph>
            <pre className="code-block">
              <code>ssh {vm.sshId}@{displayedHost}</code>
            </pre>
          </div>

          <div>
            <Title level={5}>방법 2. ~/.ssh/config 수정하고 편하게 들어가기</Title>
            <Paragraph>아래 내용을 <Text keyboard>~/.ssh/config</Text>에 추가하세요.</Paragraph>
            <pre className="code-block">
              <code>{`Host kite-${vm.name}
    HostName ${displayedHost}
    User ${vm.sshId}
    Port 22
`}</code>
            </pre>
            <Paragraph>이후에는 터미널에서 <Text keyboard>{`ssh kite-${vm.name}`}</Text> 만 입력하면 해당 VM으로 접속됩니다.</Paragraph>
          </div>
        </div>
      )}
    </Drawer>
  );
};
