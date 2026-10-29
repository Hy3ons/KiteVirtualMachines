'use client';

import { useCallback, useEffect, useState } from 'react';
import { App, Button, Skeleton } from 'antd';
import { CaretRightOutlined, CodeOutlined, PoweroffOutlined, ReloadOutlined } from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import { configApi, vmApi } from '../api';
import type { VmData } from '../api/pageData';
import type { SSHGatewaySettings, UserVm } from '../api/types';
import { ConsoleShell } from '../components/ConsoleShell';
import { useAuthStore } from '../store/useAuthStore';
import { buildSSHCommandState } from './sshGatewayCommand';
import { VmDetailTabs } from './machine/VmDetailTabs';

// VmDetail preserves VM control and connection operations within the workspace shell.
// initialData supplies the authorized SSR VM and platform settings for the current route.
export function VmDetail({ initialData }: { readonly initialData?: VmData }) {
  const { message } = App.useApp();
  const { vmName } = useParams<{ vmName: string }>();
  const router = useRouter();
  const accessLevel = useAuthStore((state) => state.accessLevel);
  const [vm, setVm] = useState<UserVm | null>(initialData?.vm ?? null);
  const [baseDomain, setBaseDomain] = useState(initialData?.config.baseDomain ?? '');
  const [secure, setSecure] = useState(initialData?.config.forceHttps ?? false);
  const [sshGateway, setSSHGateway] = useState<SSHGatewaySettings | undefined>(initialData?.config.sshGateway);
  const [loading, setLoading] = useState(!initialData);
  const [powerPending, setPowerPending] = useState(false);
  const canMutate = (accessLevel ?? 0) >= 1;

  // fetchVmDetail refreshes the selected VM and its current connection policy.
  // A failed optional config read leaves connection instructions unavailable.
  const fetchVmDetail = useCallback(async () => {
    try {
      setLoading(true);
      const [vmData, configData] = await Promise.all([vmApi.getVm(vmName), configApi.getConfig().catch(() => null)]);
      setVm(vmData.vm);
      setBaseDomain(configData?.config.baseDomain ?? '');
      setSecure(configData?.config.forceHttps ?? false);
      setSSHGateway(configData?.config.sshGateway);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'VM 정보를 불러오지 못했습니다.');
      router.push('/dashboard');
    } finally { setLoading(false); }
  }, [message, router, vmName]);

  useEffect(() => { if (!initialData) void Promise.resolve().then(fetchVmDetail); }, [initialData, fetchVmDetail]);

  // handlePower sends the desired On/Off intent to the existing Go API.
  // The local pending state prevents repeated clicks while the request is in flight.
  const handlePower = async (power: 'On' | 'Off') => {
    if (!canMutate) return;
    setPowerPending(true);
    try {
      await (power === 'On' ? vmApi.startVm(vmName) : vmApi.stopVm(vmName));
      message.success(power === 'On' ? '시작을 요청했습니다.' : '중지를 요청했습니다.');
      await fetchVmDetail();
    } catch (error) { message.error(error instanceof Error ? error.message : '전원 변경을 요청하지 못했습니다.'); }
    finally { setPowerPending(false); }
  };

  // handleDelete removes this VM through Go after the child confirmation dialog.
  // Successful deletion returns to the existing personal VM inventory.
  const handleDelete = async () => {
    if (!canMutate) return;
    try { await vmApi.deleteVm(vmName); message.success('삭제를 요청했습니다.'); router.push('/dashboard'); }
    catch (error) { message.error(error instanceof Error ? error.message : '삭제를 요청하지 못했습니다.'); }
  };

  // copySSHCommand copies the validated gateway command and reports clipboard failures.
  // Password material is never read or copied by this screen.
  const copySSHCommand = async () => {
    if (!vm) return;
    const state = buildSSHCommandState(vm.sshId, baseDomain.trim(), sshGateway);
    if (!state.ready) { message.warning(state.message); return; }
    try { await navigator.clipboard.writeText(state.command); message.success('SSH 명령어를 복사했습니다.'); }
    catch (error) { message.error(error instanceof Error ? '복사하지 못했습니다. 명령어를 직접 선택해 주세요.' : '복사에 실패했습니다.'); }
  };

  return (
    <ConsoleShell title={vm?.name ?? '가상머신'} eyebrow="가상머신" actions={<>
      <Button icon={<ReloadOutlined />} onClick={fetchVmDetail} loading={loading}>새로고침</Button>
      {vm && <><Button icon={<CodeOutlined />} disabled={vm.phase !== 'Running' || vm.delete} onClick={() => router.push(`/dashboard/kite-machine/${encodeURIComponent(vm.name)}/console`)}>시리얼 콘솔</Button><Button type={vm.phase === 'Running' ? 'default' : 'primary'} icon={vm.phase === 'Running' ? <PoweroffOutlined /> : <CaretRightOutlined />} disabled={!canMutate || vm.delete} loading={powerPending} onClick={() => handlePower(vm.phase === 'Running' ? 'Off' : 'On')}>{vm.phase === 'Running' ? '중지' : '시작'}</Button></>}
    </>}>
      {loading && !vm ? <Skeleton active paragraph={{ rows: 8 }} /> : vm && <VmDetailTabs vm={vm} secure={secure} canMutate={canMutate} sshState={buildSSHCommandState(vm.sshId, baseDomain.trim(), sshGateway)} onCopy={copySSHCommand} onDelete={handleDelete} />}
    </ConsoleShell>
  );
}
