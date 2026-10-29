import { useState, useEffect, useCallback } from 'react';
import { App as AntdApp, Form } from 'antd';
import { configApi, vmApi } from '../../api';
import { useAuthStore } from '../../store/useAuthStore';
import { LEVEL_1_FIXED_CPU, LEVEL_1_FIXED_DISK_GI, LEVEL_1_FIXED_MEMORY, LEVEL_1_VM_QUOTA } from '../userDashboardAccess';
import type { DashboardData } from '../../api/pageData';
import type { SSHGatewaySettings, VmOffer } from '../../api/types';
import type { DashboardVm, VmCreateFormValues, VmOfferClaimFormValues } from '../UserDashboardTypes';

// useDashboard owns existing API actions and overlay state; initialData seeds the SSR workspace.
export function useDashboard(initialData?: DashboardData) {
  const { message } = AntdApp.useApp();
  const { username, namespace, accessLevel } = useAuthStore();
  const safeAccessLevel = accessLevel ?? 1;
  const [vms, setVms] = useState<readonly DashboardVm[]>(initialData?.vms ?? []);
  const [offers, setOffers] = useState<readonly VmOffer[]>(initialData?.offers ?? []);
  const [loading, setLoading] = useState(!initialData);
  const [loadingOffers, setLoadingOffers] = useState(!initialData);
  const [baseDomain, setBaseDomain] = useState(initialData?.config.baseDomain ?? '');
  const [adminContact, setAdminContact] = useState(initialData?.config.adminContact ?? '');
  const [sshGateway, setSSHGateway] = useState<SSHGatewaySettings | undefined>(initialData?.config.sshGateway);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm<VmCreateFormValues>();

  const [isDrawerVisible, setIsDrawerVisible] = useState(false);
  const [selectedVm, setSelectedVm] = useState<DashboardVm | null>(null);
  const canCreateVm = safeAccessLevel >= 1 && (safeAccessLevel !== 1 || vms.length < LEVEL_1_VM_QUOTA);

  // Refreshes VM data after dashboard mutations and explicit refresh requests.
  const fetchVms = useCallback(async () => {
    try {
      setLoading(true);
      const data = await vmApi.getVms();
      setVms(data.vms || []);
    } catch {
      message.error('가상머신 목록을 불러오는데 실패했습니다.');
    } finally {
      setLoading(false);
    }
  }, [message]);

  // Loads the domain and SSH settings used by creation and connection overlays.
  const fetchPlatformConfig = useCallback(async () => {
    try {
      const data = await configApi.getConfig();
      setBaseDomain(data.config?.baseDomain || '');
      setAdminContact(data.config?.adminContact || '');
      setSSHGateway(data.config?.sshGateway);
    } catch {
      setBaseDomain('');
      setAdminContact('');
      setSSHGateway(undefined);
      message.error('접속 도메인 설정을 불러오는데 실패했습니다.');
    }
  }, [message]);

  // Refreshes available allocations after an offer is claimed.
  const fetchOffers = useCallback(async () => {
    try {
      setLoadingOffers(true);
      const data = await vmApi.getOffers();
      setOffers(data.offers || []);
    } catch {
      message.error('VM offer 목록을 불러오는데 실패했습니다.');
    } finally {
      setLoadingOffers(false);
    }
  }, [message]);

  useEffect(() => {
    if (!initialData) void Promise.resolve().then(() => Promise.all([fetchVms(), fetchPlatformConfig(), fetchOffers()]));
  }, [fetchOffers, fetchPlatformConfig, fetchVms, initialData]);

  // Requests VM power on by name from a dashboard row.
  const handleStart = async (name: string) => {
    try {
      message.loading({ content: 'Starting VM...', key: 'start' });
      await vmApi.startVm(name);
      message.success({ content: 'VM Start Requested', key: 'start' });
      fetchVms();
    } catch {
      message.error({ content: '시작 요청 실패', key: 'start' });
    }
  };

  // Requests VM power off by name from a dashboard row.
  const handleStop = async (name: string) => {
    try {
      message.loading({ content: 'Stopping VM...', key: 'stop' });
      await vmApi.stopVm(name);
      message.success({ content: 'VM Stop Requested', key: 'stop' });
      fetchVms();
    } catch {
      message.error({ content: '종료 요청 실패', key: 'stop' });
    }
  };

  // Deletes the named VM after the row confirmation.
  const handleDelete = async (name: string) => {
    try {
      message.loading({ content: 'Deleting VM...', key: 'delete' });
      await vmApi.deleteVm(name);
      message.success({ content: 'VM Delete Requested', key: 'delete' });
      fetchVms();
    } catch {
      message.error({ content: '삭제 요청 실패', key: 'delete' });
    }
  };

  // Creates a VM from validated form values, enforcing the current account quota and fixed spec.
  const handleCreate = async (values: VmCreateFormValues) => {
    if (safeAccessLevel < 1) {
      message.error(adminContact ? `VM 생성 권한이 없습니다. 관리자에게 문의하세요: ${adminContact}` : 'VM 생성 권한이 없습니다. 관리자에게 권한을 요청하세요.');
      return;
    }

    if (safeAccessLevel === 1 && vms.length >= LEVEL_1_VM_QUOTA) {
      message.error(`VM 생성 한도(최대 ${LEVEL_1_VM_QUOTA}개)를 초과했습니다. 관리자에게 권한을 요청하세요.`);
      return;
    }

    const disk = safeAccessLevel === 1 ? LEVEL_1_FIXED_DISK_GI : values.disk;
    const cpu = safeAccessLevel === 1 ? LEVEL_1_FIXED_CPU : values.cpu;
    const memory = safeAccessLevel === 1 ? LEVEL_1_FIXED_MEMORY : values.memory;
    
    try {
      setIsModalVisible(false);
      message.loading({ content: 'Creating VM...', key: 'create' });
      await vmApi.createVm({
        name: values.name,
        domainPrefix: values.domainPrefix,
        sshId: values.sshId,
        sshPassword: values.sshPassword,
        cpu,
        memory,
        disk,
        powerState: 'On'
      });
      message.success({ content: 'VM Create Requested', key: 'create' });
      form.resetFields();
      fetchVms();
    } catch (error) {
      const fallbackMessage = error instanceof Error ? error.message : '생성 실패';
      message.error({ content: fallbackMessage, key: 'create' });
    }
  };

  // Claims the selected allocation; false keeps entered credentials visible after a failure.
  const handleClaimOffer = async (offer: VmOffer, values: VmOfferClaimFormValues) => {
    try {
      message.loading({ content: 'Claiming VM offer...', key: 'claim-offer' });
      await vmApi.claimOffer(offer.name, {
        vmName: values.vmName,
        domainPrefix: values.domainPrefix,
        sshId: values.sshId,
        initialLoginPassword: values.initialLoginPassword,
        powerState: 'On',
      });
      message.success({ content: 'VM offer claim이 요청되었습니다.', key: 'claim-offer' });
      await Promise.all([fetchVms(), fetchOffers()]);
      return true;
    } catch (error) {
      const fallbackMessage = error instanceof Error ? error.message : 'VM offer claim 실패';
      message.error({ content: fallbackMessage, key: 'claim-offer' });
      return false;
    }
  };

  // Selects the row VM whose SSH instructions appear in the connection drawer.
  const showConnectGuide = (vm: DashboardVm) => {
    setSelectedVm(vm);
    setIsDrawerVisible(true);
  };

  return {
    username, namespace, safeAccessLevel, vms, offers, loading, loadingOffers,
    baseDomain, adminContact, sshGateway, isModalVisible, setIsModalVisible, form,
    isDrawerVisible, setIsDrawerVisible, selectedVm, canCreateVm, fetchVms,
    fetchOffers, handleStart, handleStop, handleDelete, handleCreate, handleClaimOffer, showConnectGuide,
  };
}
