import { useCallback, useEffect, useState } from 'react';
import { App } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { configApi, vmApi } from '../../api';
import type { VmData } from '../../api/pageData';
import type { SSHGatewaySettings, UserVm } from '../../api/types';
import { useAuthStore } from '../../store/useAuthStore';
import { buildSSHCommandState } from '../sshGatewayCommand';

import { canControlVm } from '../../domain/access';
import { vmLifecycle } from '../../domain/vmLifecycle';

// Owns VM loading, mutations, navigation and clipboard feedback for the detail screen.
export function useVmDetail(initialData: VmData | null) {
  const { message } = App.useApp();
  const { vmName } = useParams<{ vmName: string }>();
  const router = useRouter();
  const accessLevel = useAuthStore((state) => state.accessLevel);
  const [vm, setVm] = useState<UserVm | null>(initialData?.vm ?? null);
  const [baseDomain, setBaseDomain] = useState(initialData?.config.baseDomain ?? '');
  const [secure, setSecure] = useState(initialData?.config.forceHttps ?? false);
  const [sshGateway, setSSHGateway] = useState<SSHGatewaySettings | null>(initialData?.config.sshGateway ?? null);
  const [loading, setLoading] = useState(!initialData);
  const [powerPending, setPowerPending] = useState(false);
  const canMutate = canControlVm(accessLevel) && vm !== null && vmLifecycle(vm).canMutate;

  // fetchVmDetail refreshes the selected VM and its current connection policy.
  // A failed optional config read leaves connection instructions unavailable.
  const fetchVmDetail = useCallback(async () => {
    try {
      setLoading(true);
      const [vmData, configData] = await Promise.all([vmApi.getVm(vmName), configApi.getConfig().catch((error: unknown) => {
        if (!(error instanceof Error)) throw error;
        return null;
      })]);
      setVm(vmData.vm);
      setBaseDomain(configData?.config.baseDomain ?? '');
      setSecure(configData?.config.forceHttps ?? false);
      setSSHGateway(configData?.config.sshGateway ?? null);
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
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
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error; message.error(error instanceof Error ? error.message : '전원 변경을 요청하지 못했습니다.'); }
    finally { setPowerPending(false); }
  };

  // handleDelete removes this VM through Go after the child confirmation dialog.
  // Successful deletion returns to the existing personal VM inventory.
  const handleDelete = async () => {
    if (!canMutate) return;
    try { await vmApi.deleteVm(vmName); message.success('삭제를 요청했습니다.'); router.push('/dashboard'); }
    catch (error: unknown) { if (!(error instanceof Error)) throw error; message.error(error instanceof Error ? error.message : '삭제를 요청하지 못했습니다.'); }
  };

  // copySSHCommand copies the validated gateway command and reports clipboard failures.
  // Password material is never read or copied by this screen.
  const copySSHCommand = async () => {
    if (!vm || !vmLifecycle(vm).canConnect) return;
    const state = buildSSHCommandState(vm.sshId, baseDomain.trim(), sshGateway);
    if (!state.ready) { message.warning(state.message); return; }
    try { await navigator.clipboard.writeText(state.command); message.success('SSH 명령어를 복사했습니다.'); }
    catch (error: unknown) { if (!(error instanceof Error)) throw error; message.error(error instanceof Error ? '복사하지 못했습니다. 명령어를 직접 선택해 주세요.' : '복사에 실패했습니다.'); }
  };

  return { vm, baseDomain, secure, sshGateway, loading, powerPending, canMutate, fetchVmDetail, handlePower, handleDelete, copySSHCommand, router };
}
