import { useState, useEffect, useCallback } from 'react';
import { App as AntdApp, Form } from 'antd';
import { configApi, vmApi } from '../../api';
import { useAuthStore } from '../../store/useAuthStore';
import { buildVmCreatePayload, canControlVm, creationPolicy, LEVEL_1_VM_QUOTA } from '../../domain/access';
import type { DashboardData } from '../../api/pageData';
import type { SSHGatewaySettings, VmOffer } from '../../api/types';
import type { DashboardVm, VmCreateFormValues, VmOfferClaimFormValues } from '../UserDashboardTypes';

// useDashboard owns existing API actions and overlay state; initialData seeds the SSR workspace.
export function useDashboard(initialData: DashboardData | null = null) {
  const { message } = AntdApp.useApp();
  const { username, namespace, accessLevel } = useAuthStore();
  const safeAccessLevel = accessLevel ?? 0;
  const [vms, setVms] = useState<readonly DashboardVm[]>(initialData?.vms ?? []);
  const [offers, setOffers] = useState<readonly VmOffer[]>(initialData?.offers ?? []);
  const [loading, setLoading] = useState(!initialData);
  const [loadingOffers, setLoadingOffers] = useState(!initialData);
  const [baseDomain, setBaseDomain] = useState(initialData?.config.baseDomain ?? '');
  const [adminContact, setAdminContact] = useState(initialData?.config.adminContact ?? '');
  const [sshGateway, setSSHGateway] = useState<SSHGatewaySettings | null>(initialData?.config.sshGateway ?? null);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm<VmCreateFormValues>();

  const [isDrawerVisible, setIsDrawerVisible] = useState(false);
  const [selectedVm, setSelectedVm] = useState<DashboardVm | null>(null);
  const canCreateVm = creationPolicy(safeAccessLevel, vms.length) === 'allowed';

  // Refreshes VM data after dashboard mutations and explicit refresh requests.
  const fetchVms = useCallback(async () => {
    try {
      setLoading(true);
      const data = await vmApi.getVms();
      setVms(data.vms);
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error('가상머신 목록을 불러오는데 실패했습니다.');
    } finally {
      setLoading(false);
    }
  }, [message]);

  // Loads the domain and SSH settings used by creation and connection overlays.
  const fetchPlatformConfig = useCallback(async () => {
    try {
      const data = await configApi.getConfig();
      setBaseDomain(data.config.baseDomain);
      setAdminContact(data.config.adminContact);
      setSSHGateway(data.config.sshGateway);
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      setBaseDomain('');
      setAdminContact('');
      setSSHGateway(null);
      message.error('접속 도메인 설정을 불러오는데 실패했습니다.');
    }
  }, [message]);

  // Refreshes available allocations after an offer is claimed.
  const fetchOffers = useCallback(async () => {
    try {
      setLoadingOffers(true);
      const data = await vmApi.getOffers();
      setOffers(data.offers);
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
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
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
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
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
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
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error({ content: '삭제 요청 실패', key: 'delete' });
    }
  };

  // Creates a VM from validated form values, enforcing the current account quota and fixed spec.
  const handleCreate = async (values: VmCreateFormValues) => {
    const policy = creationPolicy(safeAccessLevel, vms.length);
    switch (policy) {
      case 'forbidden':
        message.error(adminContact ? `VM 생성 권한이 없습니다. 관리자에게 문의하세요: ${adminContact}` : 'VM 생성 권한이 없습니다. 관리자에게 권한을 요청하세요.');
        return;
      case 'quota':
        message.error(`VM 생성 한도(최대 ${LEVEL_1_VM_QUOTA}개)를 초과했습니다. 관리자에게 권한을 요청하세요.`);
        return;
      case 'allowed':
        break;
      default: {
        const unreachable: never = policy;
        return unreachable;
      }
    }

    try {
      message.loading({ content: 'Creating VM...', key: 'create' });
      await vmApi.createVm(buildVmCreatePayload(safeAccessLevel, values));
      setIsModalVisible(false);
      message.success({ content: 'VM Create Requested', key: 'create' });
      form.resetFields();
      fetchVms();
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
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
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
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
    username, namespace, safeAccessLevel, canMutateVm: canControlVm(accessLevel), vms, offers, loading, loadingOffers,
    baseDomain, adminContact, sshGateway, isModalVisible, setIsModalVisible, form,
    isDrawerVisible, setIsDrawerVisible, selectedVm, canCreateVm, fetchVms,
    fetchOffers, handleStart, handleStop, handleDelete, handleCreate, handleClaimOffer, showConnectGuide,
  };
}
