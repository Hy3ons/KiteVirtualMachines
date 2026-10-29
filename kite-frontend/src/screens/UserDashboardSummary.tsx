import { CloudServerOutlined, PlayCircleOutlined, PauseCircleOutlined, AppstoreOutlined } from '@ant-design/icons';
import { MetricCard } from '../components/MetricCard';
import type { DashboardVm } from './UserDashboardTypes';

type UserDashboardSummaryProps = {
  readonly accessLevel: number;
  readonly vms: readonly DashboardVm[];
  readonly quotaLimit: number;
  readonly accessDescription: string;
  readonly loading: boolean;
};

// UserDashboardSummary derives current totals from vms and describes the account's actual creation allowance.
export function UserDashboardSummary({ accessLevel, vms, quotaLimit, accessDescription, loading }: UserDashboardSummaryProps) {
  const running = vms.filter((vm) => vm.phase === 'Running').length;
  const stopped = vms.filter((vm) => vm.phase === 'Stopped').length;
  const cpu = vms.reduce((total, vm) => total + vm.cpu, 0);
  const quotaPercent = Math.min((vms.length / quotaLimit) * 100, 100);

  return (
    <section className="workspace-overview" aria-label="리소스 개요">
      <div className="workspace-allowance">
        <div className="workspace-allowance-copy"><h1>Level {accessLevel} · 리소스 권한</h1><p>{accessDescription}</p></div>
        {accessLevel === 1 ? (
          <div className="workspace-quota">
            <div><span>가상머신 할당량</span><strong>{loading ? '—' : vms.length} / {quotaLimit}</strong></div>
            <div className="workspace-quota-track" role="progressbar" aria-label="가상머신 할당량" aria-valuemin={0} aria-valuemax={quotaLimit} aria-valuenow={loading ? undefined : Math.min(vms.length, quotaLimit)}>
              <span style={{ width: `${loading ? 0 : quotaPercent}%` }} />
            </div>
          </div>
        ) : <span className="workspace-access-note">{accessLevel < 1 ? '관리자에게 생성 권한을 요청하세요' : '직접 생성 가능 · VM 개수 제한 없음'}</span>}
      </div>
      <div className="workspace-metrics">
        <MetricCard label="전체 가상머신" value={loading ? '—' : vms.length} icon={<CloudServerOutlined />} />
        <MetricCard label="실행 중" value={loading ? '—' : running} icon={<PlayCircleOutlined />} />
        <MetricCard label="중지됨" value={loading ? '—' : stopped} icon={<PauseCircleOutlined />} />
        <MetricCard label="설정된 vCPU" value={loading ? '—' : cpu} icon={<AppstoreOutlined />} />
      </div>
    </section>
  );
}
