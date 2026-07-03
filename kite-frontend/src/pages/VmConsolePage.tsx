import React, { useCallback, useEffect, useState } from 'react';
import { App as AntdApp, Avatar, Button, Card, Descriptions, Layout, Space, Spin, Tag, Typography } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { vmApi } from '../api';
import type { UserVm } from '../api/types';
import { GlobalHeader } from '../components/GlobalHeader';
import { SEO } from '../components/SEO';
import { useAuthStore } from '../store/useAuthStore';
import { useLogout } from '../hooks/useLogout';
import { VmConsoleTerminal } from './VmConsoleTerminal';

const { Content } = Layout;
const { Text, Title, Paragraph } = Typography;

const phaseColor = (phase: string): string => {
  if (phase === 'Running') return 'success';
  if (phase === 'Stopped') return 'error';
  if (phase === 'Creating' || phase === 'Terminating') return 'processing';
  return 'default';
};

export const VmConsolePage: React.FC = () => {
  const { message } = AntdApp.useApp();
  const { vmName } = useParams<{ vmName: string }>();
  const navigate = useNavigate();
  const { username, namespace, accessLevel, profileImage } = useAuthStore();
  const logout = useLogout();
  const safeAccessLevel = accessLevel ?? 1;
  const [vm, setVm] = useState<UserVm | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchVm = useCallback(async () => {
    if (!vmName) return;
    try {
      setLoading(true);
      const data = await vmApi.getVm(vmName);
      setVm(data.vm);
    } catch {
      message.error('VM 콘솔 정보를 불러오지 못했습니다.');
      navigate('/dashboard');
    } finally {
      setLoading(false);
    }
  }, [message, navigate, vmName]);

  useEffect(() => {
    void Promise.resolve().then(fetchVm);
  }, [fetchVm]);

  if (loading) {
    return (
      <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
        <GlobalHeader />
        <Content style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Spin size="large" />
        </Content>
      </Layout>
    );
  }

  if (!vm || !vmName) return null;

  const consoleEnabled = vm.phase === 'Running' && !vm.delete;

  return (
    <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
      <SEO title={`콘솔 ${vm.name} - Kite`} url={`/dashboard/kite-machine/${vm.name}/console`} />
      <GlobalHeader
        rightContent={
          <Space>
            {safeAccessLevel >= 2 && (
              <Button type="primary" ghost onClick={() => navigate('/admin/dashboard')}>Admin Console</Button>
            )}
            <Avatar src={profileImage || '/default_profile.png'} />
            <Text strong>{username}</Text>
            <Text type="secondary">({namespace})</Text>
            <Button type="text" onClick={logout}>Logout</Button>
          </Space>
        }
      />

      <Content className="console-page">
        <div className="console-page-toolbar">
          <Space wrap>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/dashboard')}>
              Dashboard
            </Button>
            <Button onClick={() => navigate(`/dashboard/kite-machine/${vm.name}`)}>
              VM detail
            </Button>
          </Space>
          <Button icon={<ReloadOutlined />} onClick={fetchVm} loading={loading}>
            Refresh
          </Button>
        </div>

        <div className="console-layout">
          <Card className="console-info-card">
            <Space orientation="vertical" size={16} className="full-width">
              <div>
                <Text type="secondary">Kite machine console</Text>
                <Title level={2} className="console-vm-title">{vm.name}</Title>
                <Space wrap>
                  <Tag color={phaseColor(vm.phase)}>{vm.phase}</Tag>
                  <Tag>{vm.currentPowerState || vm.powerState || 'Unknown'}</Tag>
                </Space>
              </div>

              <Paragraph type="secondary" className="console-copy">
                Serial console은 VM 내부 OS의 ttyS0에 직접 연결됩니다. VM 생성 시 입력한 SSH ID와 초기 비밀번호로 로그인하며, 이 비밀번호는 Kite에서 변경하거나 복구할 수 없습니다.
              </Paragraph>

              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label="Namespace">{vm.namespace}</Descriptions.Item>
                <Descriptions.Item label="CPU">{vm.cpu} Core</Descriptions.Item>
                <Descriptions.Item label="Memory">{vm.memory}</Descriptions.Item>
                <Descriptions.Item label="Disk">{vm.disk}</Descriptions.Item>
                <Descriptions.Item label="Image">{vm.image}</Descriptions.Item>
                <Descriptions.Item label="Domain">{vm.domain || '-'}</Descriptions.Item>
                <Descriptions.Item label="SSH ID">{vm.sshId || '-'}</Descriptions.Item>
                <Descriptions.Item label="DataVolume">{vm.dataVolumePhase || '-'}</Descriptions.Item>
              </Descriptions>
            </Space>
          </Card>

          <VmConsoleTerminal vmName={vm.name} enabled={consoleEnabled} mock={import.meta.env.VITE_USE_MOCK === 'true'} />
        </div>
      </Content>
    </Layout>
  );
};
