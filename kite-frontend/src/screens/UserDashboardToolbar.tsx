import { Button, Input, Select, Tooltip } from 'antd';
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';

type UserDashboardToolbarProps = {
  readonly canCreateVm: boolean;
  readonly loading: boolean;
  readonly accessLevel: number;
  readonly adminContact: string;
  readonly onRefresh: () => void;
  readonly onCreate: () => void;
  readonly query: string;
  readonly onQueryChange: (query: string) => void;
  readonly phase: string;
  readonly onPhaseChange: (phase: string) => void;
  readonly phases: readonly string[];
};

// UserDashboardToolbar exposes creation permissions and filters supplied by the workspace list.
export function UserDashboardToolbar({ canCreateVm, loading, accessLevel, adminContact, onRefresh, onCreate, query, onQueryChange, phase, onPhaseChange, phases }: UserDashboardToolbarProps) {
  const disabledReason = accessLevel < 1
    ? `VM 생성 권한이 없습니다.${adminContact ? ` 관리자 연락처: ${adminContact}` : ''}`
    : 'Level 1 VM 생성 한도에 도달했습니다.';

  return (
    <div className="workspace-list-toolbar">
      <div className="workspace-section-heading">
        <div><h2 id="instances-title">가상머신</h2></div>
        <div className="workspace-list-controls">
          <Input allowClear prefix={<SearchOutlined />} aria-label="가상머신 검색" placeholder="이름, 도메인, SSH ID 검색" value={query} onChange={(event) => onQueryChange(event.target.value)} />
          <Select aria-label="가상머신 상태 필터" value={phase} onChange={onPhaseChange} options={[{ value: 'all', label: '모든 상태' }, ...Array.from(new Set([...phases, ...(phase === 'all' ? [] : [phase])])).map((value) => ({ value, label: value }))]} />
          <Tooltip title="VM 및 할당 목록 새로고침"><Button aria-label="VM 및 할당 목록 새로고침" icon={<ReloadOutlined />} onClick={onRefresh} loading={loading} /></Tooltip>
          <Tooltip title={!canCreateVm ? disabledReason : undefined}>
            <Button type="primary" icon={<PlusOutlined />} disabled={!canCreateVm} onClick={onCreate}>가상머신 만들기</Button>
          </Tooltip>
        </div>
      </div>
      {!canCreateVm && <p className="workspace-permission-note">{disabledReason}</p>}
    </div>
  );
}
