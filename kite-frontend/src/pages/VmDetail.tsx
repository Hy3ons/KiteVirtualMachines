import React, { useState, useEffect, useCallback } from 'react';
import { SEO } from '../components/SEO';
import { vmApi } from '../api';
import { App as AntdApp, Layout, Typography, Space, Tag, Breadcrumb, Card, Tabs, Button, Descriptions, Popconfirm, Spin, Avatar, Alert } from 'antd';
import { useParams, useNavigate } from 'react-router-dom';
import { PoweroffOutlined, DeleteOutlined, CodeOutlined, DesktopOutlined, SafetyCertificateOutlined, ArrowLeftOutlined, CaretRightOutlined, CopyOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { GlobalHeader } from '../components/GlobalHeader';
import { MOCK_ENV } from '../config/mockEnv';

const { Content } = Layout;
const { Title, Text, Paragraph } = Typography;

interface VM {
  id: string;
  name: string;
  domain: string;
  phase: 'Running' | 'Stopped' | 'Creating' | 'Terminating';
  cpu: number;
  memory: string;
  disk: string;
  sshId: string;
}

export const VmDetail: React.FC = () => {
  const { message } = AntdApp.useApp();
  const { vmName } = useParams<{ vmName: string }>();
  const navigate = useNavigate();
  const { username, namespace, accessLevel, logout, profileImage } = useAuthStore();
  const safeAccessLevel = accessLevel ?? 1;
  const [vm, setVm] = useState<VM | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchVmDetail = useCallback(async () => {
    if (!vmName) return;
    try {
      setLoading(true);
      const data = await vmApi.getVm(vmName);
      setVm(data.vm);
    } catch {
      message.error('VM 정보를 불러오지 못했습니다.');
      navigate('/dashboard');
    } finally {
      setLoading(false);
    }
  }, [message, navigate, vmName]);

  useEffect(() => {
    void Promise.resolve().then(fetchVmDetail);
  }, [fetchVmDetail]);

  const handleStart = async () => {
    if (!vmName) return;
    try {
      message.loading({ content: 'Starting VM...', key: 'start' });
      await vmApi.startVm(vmName);
      message.success({ content: 'VM Start Requested', key: 'start' });
      fetchVmDetail();
    } catch {
      message.error({ content: '시작 요청 실패', key: 'start' });
    }
  };

  const handleStop = async () => {
    if (!vmName) return;
    try {
      message.loading({ content: 'Stopping VM...', key: 'stop' });
      await vmApi.stopVm(vmName);
      message.success({ content: 'VM Stop Requested', key: 'stop' });
      fetchVmDetail();
    } catch {
      message.error({ content: '종료 요청 실패', key: 'stop' });
    }
  };

  const copySSHCommand = async () => {
    if (!vm) return;
    await navigator.clipboard.writeText(`ssh ${vm.sshId}@${MOCK_ENV.BASE_DOMAIN}`);
    message.success('SSH 명령어를 복사했습니다.');
  };

  const handleDelete = async () => {
    if (!vmName) return;
    try {
      message.loading({ content: 'Deleting VM...', key: 'delete' });
      await vmApi.deleteVm(vmName);
      message.success({ content: 'VM Delete Requested', key: 'delete' });
      navigate('/dashboard');
    } catch {
      message.error({ content: '삭제 요청 실패', key: 'delete' });
    }
  };

  if (loading) {
    return (
      <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
        <GlobalHeader />
        <Content style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Spin size="large" />
        </Content>
      </Layout>
    );
  }

  if (!vm) return null;

  return (
    <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
      <SEO title={`가상머신 ${vmName || ''} - Kite`} url={`/dashboard/kite-machine/${vmName || ''}`} />
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
      
      <Content className="app-main app-main--standard vm-detail-content">
        <Breadcrumb
          style={{ marginBottom: 24 }}
          items={[
            { title: <a onClick={() => navigate('/dashboard')}>Dashboard</a> },
            { title: 'Kite Machine' },
            { title: vm.name },
          ]}
        />

        <div className="vm-detail-heading">
          <div className="vm-detail-title-group">
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/dashboard')} />
            <Title level={2} className="vm-detail-title">{vm.name}</Title>
            <Tag color={vm.phase === 'Running' ? 'success' : 'error'} style={{ fontSize: '14px', padding: '4px 8px' }}>
              {vm.phase}
            </Tag>
          </div>
          <Space>
            {vm.phase === 'Running' ? (
              <Button danger icon={<PoweroffOutlined />} onClick={handleStop}>Stop</Button>
            ) : (
              <Button type="primary" icon={<CaretRightOutlined />} onClick={handleStart}>Start</Button>
            )}
          </Space>
        </div>

        <Tabs
          defaultActiveKey="1"
          size="large"
          items={[
            {
              key: '1',
              label: <span><DesktopOutlined /> Overview</span>,
              children: (
                <Card hoverable style={{ marginTop: 16 }}>
                  <Descriptions title="가상머신 리소스 정보" bordered column={{ xxl: 3, xl: 3, lg: 3, md: 2, sm: 1, xs: 1 }}>
                    <Descriptions.Item label="VM Name">{vm.name}</Descriptions.Item>
                    <Descriptions.Item label="Namespace">{namespace}</Descriptions.Item>
                    <Descriptions.Item label="Status">{vm.phase}</Descriptions.Item>
                    <Descriptions.Item label="CPU Core">{vm.cpu} Core</Descriptions.Item>
                    <Descriptions.Item label="Memory">{vm.memory}</Descriptions.Item>
                    <Descriptions.Item label="Disk">{vm.disk}</Descriptions.Item>
                  </Descriptions>
                  
                  <div style={{ marginTop: 40 }}>
                    <Title level={4}>네트워크 및 서비스 연동</Title>
                    <Paragraph type="secondary">Kite 컨트롤러가 할당한 도메인과 내부 ClusterIP 서비스를 통해 연결됩니다.</Paragraph>
                    <Descriptions bordered column={1}>
                      <Descriptions.Item label="Public Domain">
                        <a href={`http://${vm.domain}`} target="_blank" rel="noreferrer">http://{vm.domain}</a>
                      </Descriptions.Item>
                      <Descriptions.Item label="SSH User">
                        <Text strong>{vm.sshId}</Text>
                      </Descriptions.Item>
                    </Descriptions>
                  </div>
                </Card>
              )
            },
            {
              key: '2',
              label: <span><CodeOutlined /> SSH Connection</span>,
              children: (
                <Card hoverable style={{ marginTop: 16 }}>
                  <Title level={4}>SSH 접속 방법</Title>
                  <Paragraph>Kite가 설치된 서버에 VM 생성 시 입력한 계정으로 접속합니다.</Paragraph>
                  {vm.phase !== 'Running' && (
                    <Alert
                      message="VM이 실행 중이 아닙니다."
                      description="SSH 접속은 VM이 Running 상태가 된 뒤에 사용할 수 있습니다."
                      type="warning"
                      showIcon
                      style={{ marginBottom: 24 }}
                    />
                  )}
                  
                  <div style={{ background: '#f5f5f5', padding: '16px', borderRadius: 0, marginBottom: '24px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, alignItems: 'center' }}>
                      <Text strong>한 줄 명령어로 바로 접속하기</Text>
                      <Button icon={<CopyOutlined />} onClick={copySSHCommand}>Copy</Button>
                    </div>
                    <div style={{ marginTop: '8px', fontFamily: 'monospace', background: '#2d2d2d', color: '#fff', padding: '12px', borderRadius: 0 }}>
                      ssh {vm.sshId}@{MOCK_ENV.BASE_DOMAIN}
                    </div>
                  </div>
                </Card>
              )
            },
            {
              key: '3',
              label: <span><SafetyCertificateOutlined /> Settings</span>,
              children: (
                <Card hoverable style={{ marginTop: 16, border: '1px solid #ffccc7' }}>
                  <Title level={4} type="danger">Danger Zone</Title>
                  <Paragraph>이 작업은 되돌릴 수 없으며, 가상머신의 모든 데이터가 영구적으로 삭제됩니다.</Paragraph>
                  <Popconfirm title="정말 삭제하시겠습니까? 모든 데이터가 유실됩니다." onConfirm={handleDelete} okText="삭제" okButtonProps={{ danger: true }}>
                    <Button danger type="primary" icon={<DeleteOutlined />}>Delete Virtual Machine</Button>
                  </Popconfirm>
                </Card>
              )
            }
          ]}
        />
      </Content>
    </Layout>
  );
};
