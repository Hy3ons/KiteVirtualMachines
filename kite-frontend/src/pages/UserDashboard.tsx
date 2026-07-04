import React, { useState, useEffect, useCallback } from 'react';
import { SEO } from '../components/SEO';
import { configApi, vmApi } from '../api';
import { App as AntdApp, Layout, Typography, Button, Table, Space, Form, Avatar, Empty } from 'antd';
import { PlusOutlined, LogoutOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { useLogout } from '../hooks/useLogout';
import { useNavigate } from 'react-router-dom';
import { GlobalHeader } from '../components/GlobalHeader';
import { UserDashboardSummary } from './UserDashboardSummary';
import { VmCreateModal } from './VmCreateModal';
import { VmConnectionDrawer } from './VmConnectionDrawer';
import { UserVmOffersSection } from './UserVmOffersSection';
import { UserDashboardToolbar } from './UserDashboardToolbar';
import { createUserDashboardColumns } from './UserDashboardColumns';
import { LEVEL_1_FIXED_CPU, LEVEL_1_FIXED_DISK_GI, LEVEL_1_FIXED_MEMORY, LEVEL_1_VM_QUOTA, MIN_DISK_GI, getAccessLevelDescription } from './userDashboardAccess';
import type { VmOffer } from '../api/types';
import type { DashboardVm, VmCreateFormValues, VmOfferClaimFormValues } from './UserDashboardTypes';

const { Content } = Layout;
const { Text } = Typography;

export const UserDashboard: React.FC = () => {
  const { message } = AntdApp.useApp();
  const { username, namespace, accessLevel, profileImage } = useAuthStore();
  const logout = useLogout();
  const safeAccessLevel = accessLevel ?? 1;
  const navigate = useNavigate();
  const [vms, setVms] = useState<DashboardVm[]>([]);
  const [offers, setOffers] = useState<VmOffer[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingOffers, setLoadingOffers] = useState(true);
  const [baseDomain, setBaseDomain] = useState('');
  const [adminContact, setAdminContact] = useState('');
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();

  const [isDrawerVisible, setIsDrawerVisible] = useState(false);
  const [selectedVm, setSelectedVm] = useState<DashboardVm | null>(null);
  const canCreateVm = safeAccessLevel >= 1 && (safeAccessLevel !== 1 || vms.length < LEVEL_1_VM_QUOTA);

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

  const fetchPlatformConfig = useCallback(async () => {
    try {
      const data = await configApi.getConfig();
      setBaseDomain(data.config?.baseDomain || '');
      setAdminContact(data.config?.adminContact || '');
    } catch {
      setBaseDomain('');
      setAdminContact('');
      message.error('접속 도메인 설정을 불러오는데 실패했습니다.');
    }
  }, [message]);

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
    void Promise.resolve().then(() => Promise.all([fetchVms(), fetchPlatformConfig(), fetchOffers()]));
  }, [fetchOffers, fetchPlatformConfig, fetchVms]);

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
    } catch (error) {
      const fallbackMessage = error instanceof Error ? error.message : 'VM offer claim 실패';
      message.error({ content: fallbackMessage, key: 'claim-offer' });
    }
  };

  const showConnectGuide = (vm: DashboardVm) => {
    setSelectedVm(vm);
    setIsDrawerVisible(true);
  };

  const columns = createUserDashboardColumns({
    onNavigateToDetail: (name) => navigate(`/dashboard/kite-machine/${name}`),
    onConnect: showConnectGuide,
    onOpenConsole: (name) => navigate(`/dashboard/kite-machine/${name}/console`),
    onStart: handleStart,
    onStop: handleStop,
    onDelete: handleDelete,
    canMutateVm: safeAccessLevel >= 1
  });

  return (
    <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
      <SEO title="내 대시보드 - Kite" url="/dashboard" />
      <GlobalHeader 
        rightContent={
          <Space>
            {safeAccessLevel >= 2 && (
              <Button type="primary" ghost onClick={() => navigate('/admin/dashboard')}>Admin Console</Button>
            )}
            <Avatar src={profileImage || '/default_profile.png'} />
            <Text strong>{username}</Text>
            <Text type="secondary">({namespace})</Text>
            <Button type="text" icon={<LogoutOutlined />} onClick={logout}>Logout</Button>
          </Space>
        }
      />
      
      <Content className="dashboard-content">
        <UserDashboardSummary
          username={username}
          namespace={namespace}
          accessLevel={safeAccessLevel}
          vms={vms}
          quotaLimit={LEVEL_1_VM_QUOTA}
          accessDescription={getAccessLevelDescription(safeAccessLevel)}
        />

        <UserDashboardToolbar
          canCreateVm={canCreateVm}
          loading={loading}
          accessLevel={safeAccessLevel}
          adminContact={adminContact}
          onRefresh={fetchVms}
          onCreate={() => setIsModalVisible(true)}
        />
        
        <Table 
          columns={columns} 
          dataSource={vms} 
          rowKey="id" 
          loading={loading}
          pagination={false}
          scroll={{ x: 1430 }}
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={safeAccessLevel < 1 ? `VM 생성 권한이 없습니다.${adminContact ? ` 관리자 연락처: ${adminContact}` : ''}` : '아직 생성된 VM이 없습니다.'}
              >
                {canCreateVm && (
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => setIsModalVisible(true)}>
                    Create VM
                  </Button>
                )}
              </Empty>
            )
          }}
        />

        <UserVmOffersSection
          offers={offers}
          loading={loadingOffers}
          onClaim={handleClaimOffer}
        />
      </Content>

      <VmCreateModal
        open={isModalVisible}
        onCancel={() => setIsModalVisible(false)}
        onCreate={handleCreate}
        form={form}
        canCreateVm={canCreateVm}
        accessLevel={safeAccessLevel}
        fixedCpu={LEVEL_1_FIXED_CPU}
        fixedMemory={LEVEL_1_FIXED_MEMORY}
        fixedDiskGi={LEVEL_1_FIXED_DISK_GI}
        minDiskGi={MIN_DISK_GI}
        adminContact={adminContact}
        baseDomain={baseDomain}
      />

      <VmConnectionDrawer
        open={isDrawerVisible}
        vm={selectedVm}
        baseDomain={baseDomain}
        onClose={() => setIsDrawerVisible(false)}
      />
    </Layout>
  );
};
