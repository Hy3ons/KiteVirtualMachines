'use client';

import { Button, Skeleton } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import dynamic from 'next/dynamic';
import type { UserVm } from '../api/types';
import { ConsoleShell } from '../components/ConsoleShell';
import { StatusBadge } from '../components/StatusBadge';
import { useVmConsole } from './machine/useVmConsole';
import { vmLifecycle } from '../domain/vmLifecycle';
import { USE_MOCK_API } from '../config/debug';

const VmConsoleTerminal = dynamic(() => import('./VmConsoleTerminal').then((module) => module.VmConsoleTerminal), { ssr: false });

// VmConsolePage keeps the serial connection in a client-only xterm leaf.
// initialVm is authorized SSR metadata; refresh and navigation retain the existing Go endpoints.
export function VmConsolePage({ initialVm = null }: { readonly initialVm?: UserVm | null }) {
  const { vm, vmName, router, loading, fetchVm } = useVmConsole(initialVm);

  return (
    <ConsoleShell title={vm?.name ?? vmName} eyebrow="가상머신 / 콘솔" actions={<>
      {vm && <span className="serial-page-status"><StatusBadge phase={vm.phase} /></span>}
      <Button icon={<ArrowLeftOutlined />} onClick={() => router.push(vm ? `/dashboard/kite-machine/${encodeURIComponent(vm.name)}` : '/dashboard')}>VM 상세</Button><Button icon={<ReloadOutlined />} onClick={fetchVm} loading={loading}>새로고침</Button>
    </>}>
      {loading && !vm ? <Skeleton active paragraph={{ rows: 8 }} /> : vm && <div className="serial-workspace">
        <VmConsoleTerminal vmName={vm.name} enabled={vmLifecycle(vm).canConnect} mock={USE_MOCK_API} />
        <div className="serial-support"><section className="machine-surface"><h2>연결 정보</h2><dl className="machine-properties"><div><dt>SSH 사용자</dt><dd>{vm.sshId || '—'}</dd></div><div><dt>도메인</dt><dd>{vm.domain || '—'}</dd></div><div><dt>이미지</dt><dd>{vm.image}</dd></div><div><dt>전원</dt><dd>{vm.currentPowerState || vm.powerState}</dd></div></dl></section>
          <section className="machine-surface"><h2>시작하기 전에</h2><p>시리얼 콘솔은 VM의 ttyS0에 연결됩니다. VM을 만들 때 지정한 SSH ID와 초기 비밀번호로 로그인하세요.</p><p className="machine-section-caption">{vm.cpu} vCPU · {vm.memory} 메모리 · {vm.disk} 디스크</p><p className="machine-section-caption">비밀번호는 Kite에서 변경하거나 복구할 수 없습니다. 연결이 끊어지면 콘솔 상단에서 다시 연결하세요.</p></section></div>
      </div>}
    </ConsoleShell>
  );
}
