'use client';

import { Button, Skeleton } from 'antd';
import { CaretRightOutlined, CodeOutlined, PoweroffOutlined, ReloadOutlined } from '@ant-design/icons';
import type { VmData } from '../api/pageData';
import { ConsoleShell } from '../components/ConsoleShell';
import { buildSSHCommandState } from './sshGatewayCommand';
import { useVmDetail } from './machine/useVmDetail';
import { vmLifecycle } from '../domain/vmLifecycle';
import { VmDetailTabs } from './machine/VmDetailTabs';

// VmDetail preserves VM control and connection operations within the workspace shell.
// initialData supplies the authorized SSR VM and platform settings for the current route.
export function VmDetail({ initialData = null }: { readonly initialData?: VmData | null }) {
  const { vm, baseDomain, secure, sshGateway, loading, powerPending, canMutate, fetchVmDetail, handlePower, handleDelete, copySSHCommand, router } = useVmDetail(initialData);

  return (
    <ConsoleShell title={vm?.name ?? '가상머신'} eyebrow="가상머신" actions={<>
      <Button icon={<ReloadOutlined />} onClick={fetchVmDetail} loading={loading}>새로고침</Button>
      {vm && <><Button icon={<CodeOutlined />} disabled={!vmLifecycle(vm).canConnect} onClick={() => router.push(`/dashboard/kite-machine/${encodeURIComponent(vm.name)}/console`)}>시리얼 콘솔</Button><Button type={vm.phase === 'Running' ? 'default' : 'primary'} icon={vm.phase === 'Running' ? <PoweroffOutlined /> : <CaretRightOutlined />} disabled={!canMutate || vm.delete} loading={powerPending} onClick={() => handlePower(vmLifecycle(vm).nextPower)}>{vm.phase === 'Running' ? '중지' : '시작'}</Button></>}
    </>}>
      {loading && !vm ? <Skeleton active paragraph={{ rows: 8 }} /> : vm && <VmDetailTabs vm={vm} secure={secure} canMutate={canMutate} sshState={buildSSHCommandState(vm.sshId, baseDomain.trim(), sshGateway)} onCopy={copySSHCommand} onDelete={handleDelete} />}
    </ConsoleShell>
  );
}
