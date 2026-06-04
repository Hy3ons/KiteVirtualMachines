import React, { useState, useEffect } from 'react';
import { SEO } from '../components/SEO';
import { vmApi } from '../api';
import { Layout, Typography, Space, Tag, Breadcrumb, Card, Tabs, Button, Descriptions, message, Popconfirm, Spin, Avatar } from 'antd';
import { useParams, useNavigate } from 'react-router-dom';
import { PoweroffOutlined, DeleteOutlined, CodeOutlined, DesktopOutlined, SafetyCertificateOutlined, ArrowLeftOutlined } from '@ant-design/icons';
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
  const { vmName } = useParams<{ vmName: string }>();
  const navigate = useNavigate();
  const { username, namespace, accessLevel, logout, profileImage } = useAuthStore();
  const safeAccessLevel = accessLevel ?? 1;
  const [vm, setVm] = useState<VM | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchVmDetail = async () => {
    if (!vmName) return;
    try {
      setLoading(true);
      const data = await vmApi.getVm(vmName);
      setVm(data.vm);
    } catch (error) {
      message.error('VM 정보를 불러오지 못했습니다.');
      navigate('/dashboard');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchVmDetail();
  }, [vmName]);

  const handleStop = async () => {
    if (!vmName) return;
    try {
      message.loading({ content: 'Stopping VM...', key: 'stop' });
      await vmApi.stopVm(vmName);
      message.success({ content: 'VM Stop Requested', key: 'stop' });
      fetchVmDetail();
    } catch (error) {
      message.error({ content: '종료 요청 실패', key: 'stop' });
    }
  };

  const handleDelete = async () => {
    if (!vmName) return;
    try {
      message.loading({ content: 'Deleting VM...', key: 'delete' });
      await vmApi.deleteVm(vmName);
      message.success({ content: 'VM Delete Requested', key: 'delete' });
      navigate('/dashboard');
    } catch (error) {
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
      
      <Content style={{ padding: '40px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>
        <Breadcrumb style={{ marginBottom: 24 }}>
          <Breadcrumb.Item><a onClick={() => navigate('/dashboard')}>Dashboard</a></Breadcrumb.Item>
          <Breadcrumb.Item>Kite Machine</Breadcrumb.Item>
          <Breadcrumb.Item>{vm.name}</Breadcrumb.Item>
        </Breadcrumb>

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 32 }}>
          <Space align="center" size="middle">
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/dashboard')} />
            <Title level={2} style={{ margin: 0 }}>{vm.name}</Title>
            <Tag color={vm.phase === 'Running' ? 'success' : 'error'} style={{ fontSize: '14px', padding: '4px 8px' }}>
              {vm.phase}
            </Tag>
          </Space>
          <Space>
            {vm.phase === 'Running' ? (
              <Button danger icon={<PoweroffOutlined />} onClick={handleStop}>Stop</Button>
            ) : (
              <Button type="primary" style={{ backgroundColor: '#52c41a' }} onClick={() => setVm({ ...vm, phase: 'Running' })}>Start</Button>
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
                  
                  <div style={{ background: '#f5f5f5', padding: '16px', borderRadius: '4px', marginBottom: '24px' }}>
                    <Text strong>한 줄 명령어로 바로 접속하기 (가장 간단함)</Text>
                    <div style={{ marginTop: '8px', fontFamily: 'monospace', background: '#2d2d2d', color: '#fff', padding: '12px', borderRadius: '4px' }}>
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
