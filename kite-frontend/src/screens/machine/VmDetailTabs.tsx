import { Alert, Button, Popconfirm, Space, Tabs } from 'antd';
import { CloudServerOutlined, CodeOutlined, CopyOutlined, DeleteOutlined, SettingOutlined } from '@ant-design/icons';
import { StatusBadge } from '../../components/StatusBadge';
import type { UserVm } from '../../api/types';
import type { buildSSHCommandState } from '../sshGatewayCommand';

type VmDetailTabsProps = {
  readonly vm: UserVm;
  readonly secure: boolean;
  readonly canMutate: boolean;
  readonly sshState: ReturnType<typeof buildSSHCommandState>;
  readonly onCopy: () => Promise<void>;
  readonly onDelete: () => Promise<void>;
};

// VmDetailTabs groups observed VM data, SSH instructions and destructive settings.
// vm is API-backed metadata; callbacks keep mutations and clipboard feedback in the parent.
export function VmDetailTabs({ vm, secure, canMutate, sshState, onCopy, onDelete }: VmDetailTabsProps) {
  return (
    <Tabs className="machine-tabs" defaultActiveKey="overview" items={[
      { key: 'overview', label: <span><CloudServerOutlined /> 개요</span>, children: (
        <div className="machine-overview">
          <section className="machine-surface"><div className="machine-section-heading"><h2>리소스 구성</h2><span className="machine-section-caption">Virtual machine</span></div>
            <div className="machine-specs"><div><span>vCPU</span><strong>{vm.cpu}<small>cores</small></strong></div><div><span>메모리</span><strong>{vm.memory}</strong></div><div><span>디스크</span><strong>{vm.disk}</strong></div></div>
            <dl className="machine-properties"><div><dt>이름</dt><dd>{vm.name}</dd></div><div><dt>이미지</dt><dd>{vm.image}</dd></div><div><dt>원하는 전원 상태</dt><dd>{vm.powerState}</dd></div><div><dt>현재 전원 상태</dt><dd>{vm.currentPowerState || '확인 중'}</dd></div></dl>
          </section>
          <section className="machine-surface"><div className="machine-section-heading"><h2>연결 및 상태</h2><StatusBadge phase={vm.phase} /></div>
            <dl className="machine-properties"><div><dt>공개 도메인</dt><dd>{vm.domain ? <a href={`${secure ? 'https' : 'http'}://${vm.domain}`} target="_blank" rel="noreferrer">{vm.domain}</a> : '아직 할당되지 않음'}</dd></div><div><dt>SSH 사용자</dt><dd className="technical-text">{vm.sshId || '—'}</dd></div><div><dt>볼륨 상태</dt><dd>{vm.dataVolumePhase || '확인 중'}</dd></div><div><dt>볼륨 진행률</dt><dd>{vm.dataVolumeProgress || '—'}</dd></div></dl>
            {vm.dataVolumeMessage && <p className="machine-volume-note">{vm.dataVolumeMessage}</p>}
          </section>
        </div>
      ) },
      { key: 'ssh', label: <span><CodeOutlined /> SSH 연결</span>, children: (
        <section className="machine-surface machine-ssh"><div className="machine-section-heading"><h2>터미널에서 연결하기</h2><span className="machine-section-caption">SSH gateway</span></div>
          <p>VM을 만들 때 지정한 SSH 계정과 초기 비밀번호를 사용하세요.</p>
          {!sshState.ready && <Alert className="form-alert" title="SSH gateway 설정 대기" description={sshState.message} type="warning" showIcon />}
          {vm.phase !== 'Running' && <Alert className="form-alert" title="VM이 실행 중이 아닙니다." description="Running 상태가 된 후 SSH에 연결할 수 있습니다." type="warning" showIcon />}
          {sshState.ready && <div className="machine-command"><div><span>SSH 명령어</span><Button size="small" icon={<CopyOutlined />} onClick={onCopy}>복사</Button></div><code>{sshState.command}</code></div>}
          <p className="machine-section-caption">초기 비밀번호는 Kite에서 확인하거나 복구할 수 없습니다.</p>
        </section>
      ) },
      { key: 'settings', label: <span><SettingOutlined /> 설정</span>, children: (
        <section className="machine-surface machine-danger"><h2>가상머신 삭제</h2><p>가상머신과 연결된 데이터가 영구적으로 삭제됩니다. 필요한 데이터를 먼저 백업하세요.</p>
          <Space><Popconfirm title="가상머신을 삭제하시겠습니까?" description="모든 데이터가 영구적으로 삭제됩니다." onConfirm={onDelete} okText="삭제" cancelText="취소" okButtonProps={{ danger: true }} disabled={!canMutate}><Button danger icon={<DeleteOutlined />} disabled={!canMutate}>가상머신 삭제</Button></Popconfirm></Space>
        </section>
      ) },
    ]} />
  );
}
