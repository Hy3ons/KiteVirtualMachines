import React, { useState, useEffect } from 'react';
import { SEO } from '../components/SEO';
import { vmApi } from '../api';
import { Layout, Typography, Button, Table, Space, Tag, Modal, Form, Input, InputNumber, message, Popconfirm, Drawer, Alert, Row, Col, Card, Avatar, Statistic, Progress, Empty, Tooltip } from 'antd';
import { PlusOutlined, PoweroffOutlined, CaretRightOutlined, DeleteOutlined, LogoutOutlined, CodeOutlined, UserOutlined, ReloadOutlined, CloudServerOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { useNavigate } from 'react-router-dom';
import { GlobalHeader } from '../components/GlobalHeader';
import { MOCK_ENV } from '../config/mockEnv';

const { Content } = Layout;
const { Title, Text, Paragraph } = Typography;

const LEVEL_1_FIXED_CPU = 2;
const LEVEL_1_FIXED_MEMORY = '4Gi';
const LEVEL_1_FIXED_DISK_GI = 20;
const MIN_DISK_GI = 20;
const LEVEL_1_VM_QUOTA = MOCK_ENV.MAX_VM_QUOTA_LEVEL_1;

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

export const UserDashboard: React.FC = () => {
  const { username, namespace, accessLevel, logout, profileImage } = useAuthStore();
  const safeAccessLevel = accessLevel ?? 1;
  const navigate = useNavigate();
  const [vms, setVms] = useState<VM[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();

  // Connection Drawer State
  const [isDrawerVisible, setIsDrawerVisible] = useState(false);
  const [selectedVm, setSelectedVm] = useState<VM | null>(null);
  const runningVmCount = vms.filter((vm) => vm.phase === 'Running').length;
  const stoppedVmCount = vms.filter((vm) => vm.phase === 'Stopped').length;
  const pendingVmCount = vms.filter((vm) => vm.phase === 'Creating' || vm.phase === 'Terminating').length;
  const quotaPercent = safeAccessLevel === 1 ? Math.min(Math.round((vms.length / LEVEL_1_VM_QUOTA) * 100), 100) : 0;
  const canCreateVm = safeAccessLevel >= 1 && (safeAccessLevel !== 1 || vms.length < LEVEL_1_VM_QUOTA);

  // 권한별 설명 텍스트
  const getAccessLevelDescription = (level: number) => {
    if (level === 0) return 'VM 생성 권한이 없는 계정입니다. 사용이 필요하면 관리자에게 권한을 요청하세요.';
    if (level === 1) return '일반 계정입니다. VM은 최대 3개까지 생성할 수 있고 스펙은 CPU 2, RAM 4Gi, Disk 20Gi로 고정됩니다.';
    if (level === 2) return '매니저 계정입니다. 일반 권한을 포함하며, 타 유저의 VM 상태 제어 및 관리가 가능합니다.';
    if (level >= 3) return '최고 관리자 계정입니다. 매니저 권한을 포함하며, Host Setup 및 클러스터 인프라 전역을 통제할 수 있습니다.';
    return '';
  };

  const fetchVms = async () => {
    try {
      setLoading(true);
      const data = await vmApi.getVms();
      setVms(data.vms || []);
    } catch (error) {
      message.error('가상머신 목록을 불러오는데 실패했습니다.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchVms();
  }, []);

  const handleStart = async (name: string) => {
    try {
      message.loading({ content: 'Starting VM...', key: 'start' });
      await vmApi.startVm(name);
      message.success({ content: 'VM Start Requested', key: 'start' });
      fetchVms();
    } catch (error) {
      message.error({ content: '시작 요청 실패', key: 'start' });
    }
  };

  const handleStop = async (name: string) => {
    try {
      message.loading({ content: 'Stopping VM...', key: 'stop' });
      await vmApi.stopVm(name);
      message.success({ content: 'VM Stop Requested', key: 'stop' });
      fetchVms();
    } catch (error) {
      message.error({ content: '종료 요청 실패', key: 'stop' });
    }
  };

  const handleDelete = async (name: string) => {
    try {
      message.loading({ content: 'Deleting VM...', key: 'delete' });
      await vmApi.deleteVm(name);
      message.success({ content: 'VM Delete Requested', key: 'delete' });
      fetchVms();
    } catch (error) {
      message.error({ content: '삭제 요청 실패', key: 'delete' });
    }
  };

  const handleCreate = async (values: any) => {
    if (safeAccessLevel < 1) {
      message.error('VM 생성 권한이 없습니다. 관리자에게 권한을 요청하세요.');
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
    } catch (error: any) {
      message.error({ content: error.response?.data?.message || '생성 실패', key: 'create' });
    }
  };

  const showConnectGuide = (vm: VM) => {
    setSelectedVm(vm);
    setIsDrawerVisible(true);
  };

  const columns = [
    { 
      title: 'Name', 
      dataIndex: 'name', 
      key: 'name', 
      render: (text: string) => (
        <a onClick={() => navigate(`/dashboard/kite-machine/${text}`)} style={{ fontWeight: 'bold', color: '#8B7355', textDecoration: 'underline' }}>
          {text}
        </a>
      ) 
    },
    { title: 'Domain', dataIndex: 'domain', key: 'domain' },
    { title: 'SSH ID', dataIndex: 'sshId', key: 'sshId' },
    { title: 'Status', dataIndex: 'phase', key: 'phase', render: (phase: string) => {
      let color = 'default';
      if (phase === 'Running') color = 'success';
      if (phase === 'Stopped') color = 'error';
      if (phase === 'Creating' || phase === 'Terminating') color = 'processing';
      return <Tag color={color}>{phase}</Tag>;
    }},
    { title: 'CPU', dataIndex: 'cpu', key: 'cpu' },
    { title: 'Memory', dataIndex: 'memory', key: 'memory' },
    { title: 'Disk', dataIndex: 'disk', key: 'disk' },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: VM) => (
        <Space size="middle">
          <Button type="text" icon={<CodeOutlined />} onClick={() => showConnectGuide(record)}>Connect</Button>
          {record.phase !== 'Running' && <Button type="text" icon={<CaretRightOutlined style={{ color: '#52c41a' }} />} onClick={() => handleStart(record.name)} />}
          {record.phase === 'Running' && <Button type="text" icon={<PoweroffOutlined style={{ color: '#d9363e' }} />} onClick={() => handleStop(record.name)}>Stop</Button>}
          <Popconfirm title="정말 이 VM을 삭제하시겠습니까? 데이터가 모두 날아갑니다." onConfirm={() => handleDelete(record.name)}>
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    },
  ];

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
      
      <Content style={{ padding: '40px', maxWidth: '1300px', margin: '0 auto', width: '100%' }}>
        <Row gutter={[24, 24]} style={{ marginBottom: 32 }}>
          <Col xs={24} lg={14}>
            <Card hoverable>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <Title level={4} style={{ margin: 0, color: '#333', display: 'flex', alignItems: 'center' }}>
                    <UserOutlined style={{ marginRight: 8, color: '#C9B59C' }} /> 
                    {username}님의 프로필
                  </Title>
                  <Text type="secondary" style={{ marginTop: 6, display: 'block' }}>
                    {getAccessLevelDescription(safeAccessLevel)}
                  </Text>
                </div>
                <Space size="large">
                  <div style={{ textAlign: 'center' }}>
                    <Text type="secondary" style={{ fontSize: '12px' }}>NAMESPACE</Text>
                    <div style={{ fontSize: '16px', fontWeight: 'bold' }}>{namespace}</div>
                  </div>
                  <div style={{ textAlign: 'center', borderLeft: '1px solid #eee', paddingLeft: '24px' }}>
                    <Text type="secondary" style={{ fontSize: '12px' }}>ACCESS LEVEL</Text>
                    <div style={{ fontSize: '16px', fontWeight: 'bold' }}>Level {safeAccessLevel}</div>
                  </div>
                  <div style={{ textAlign: 'center', borderLeft: '1px solid #eee', paddingLeft: '24px' }}>
                    <Text type="secondary" style={{ fontSize: '12px' }}>VM QUOTA</Text>
                    <div style={{ fontSize: '16px', fontWeight: 'bold', color: '#C9B59C' }}>
                      {safeAccessLevel === 0 ? 'No Access' : safeAccessLevel === 1 ? 'Max 3 VMs' : 'Unlimited'}
                    </div>
                  </div>
                </Space>
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={8} lg={4}>
            <Card hoverable>
              <Statistic title="VMs" value={vms.length} prefix={<CloudServerOutlined />} valueStyle={{ color: '#8B7355' }} />
            </Card>
          </Col>
          <Col xs={24} sm={8} lg={3}>
            <Card hoverable>
              <Statistic title="Running" value={runningVmCount} valueStyle={{ color: '#7A9C74' }} />
            </Card>
          </Col>
          <Col xs={24} sm={8} lg={3}>
            <Card hoverable>
              <Statistic title={pendingVmCount > 0 ? 'Pending' : 'Stopped'} value={pendingVmCount > 0 ? pendingVmCount : stoppedVmCount} valueStyle={{ color: pendingVmCount > 0 ? '#D4A373' : '#C86B6B' }} />
            </Card>
          </Col>
        </Row>

        {safeAccessLevel === 1 && (
          <Card style={{ marginBottom: 24 }}>
            <Space direction="vertical" style={{ width: '100%' }} size={8}>
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16 }}>
                <Text strong>Level 1 VM quota</Text>
                <Text type="secondary">{vms.length} / {LEVEL_1_VM_QUOTA}</Text>
              </div>
              <Progress percent={quotaPercent} strokeColor="#8B7355" showInfo={false} />
              <Text type="secondary">CPU 2, RAM 4Gi, Disk 20Gi 스펙으로만 생성됩니다.</Text>
            </Space>
          </Card>
        )}

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 16, marginBottom: '24px' }}>
          <Title level={2} style={{ margin: 0 }}>My Virtual Machines</Title>
          <Space>
            <Tooltip title="목록 새로고침">
              <Button icon={<ReloadOutlined />} onClick={fetchVms} loading={loading} />
            </Tooltip>
            <Tooltip title={!canCreateVm ? (safeAccessLevel < 1 ? 'VM 생성 권한이 없습니다.' : 'Level 1 VM 생성 한도에 도달했습니다.') : ''}>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                size="large"
                disabled={!canCreateVm}
                onClick={() => setIsModalVisible(true)}
              >
                Create VM
              </Button>
            </Tooltip>
          </Space>
        </div>
        
        <Table 
          columns={columns} 
          dataSource={vms} 
          rowKey="id" 
          loading={loading}
          pagination={false}
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={safeAccessLevel < 1 ? 'VM 생성 권한이 없습니다.' : '아직 생성된 VM이 없습니다.'}
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
      </Content>

      <Modal 
        title={<Title level={4} style={{ margin: 0 }}>Create Virtual Machine</Title>} 
        open={isModalVisible} 
        onCancel={() => setIsModalVisible(false)}
        footer={null}
        centered
      >
        <Form form={form} layout="vertical" onFinish={handleCreate} style={{ marginTop: 24 }}>
          {safeAccessLevel === 0 && (
            <Alert
              message="VM 생성 권한 없음"
              description="Level 0 계정은 VM을 생성할 수 없습니다. 관리자에게 권한을 요청하세요."
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />
          )}
          {safeAccessLevel === 1 && (
            <Alert
              message="일반 계정 VM 스펙 고정"
              description={`Level 1 계정은 CPU ${LEVEL_1_FIXED_CPU}, RAM ${LEVEL_1_FIXED_MEMORY}, Disk ${LEVEL_1_FIXED_DISK_GI}Gi로 생성됩니다.`}
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />
          )}

          <Form.Item name="name" label="VM Name" rules={[{ required: true, message: '영문 소문자/숫자 조합의 이름을 입력하세요.' }]}>
            <Input placeholder="my-ubuntu-vm" />
          </Form.Item>
          
          <Form.Item name="domainPrefix" label="Domain Prefix">
            <Input addonAfter={`.${MOCK_ENV.BASE_DOMAIN} (예시)`} placeholder="my-web" />
          </Form.Item>
          
          <Form.Item name="sshId" label="SSH User ID" rules={[{ required: true, message: '접속용 ID를 입력하세요.' }]}>
            <Input placeholder="ubuntu" />
          </Form.Item>
          
          <Form.Item name="sshPassword" label="SSH Password" rules={[{ required: true, message: '비밀번호를 입력하세요.' }]}>
            <Input.Password placeholder="Strong password" />
          </Form.Item>
          
          <Form.Item name="disk" label="Disk Size (Gi)" initialValue={safeAccessLevel === 1 ? LEVEL_1_FIXED_DISK_GI : MIN_DISK_GI} rules={[{ required: true }]}>
            <InputNumber min={MIN_DISK_GI} max={100} disabled={safeAccessLevel === 1} style={{ width: '100%' }} />
          </Form.Item>

          <div style={{ backgroundColor: '#EFE9E3', padding: '12px', marginBottom: '24px', fontSize: '13px', color: '#555' }}>
            {safeAccessLevel === 1
              ? '* Level 1 계정은 CPU 2, Memory 4Gi, Disk 20Gi, OS Image ubuntu-22.04로 고정 생성됩니다.'
              : '* 최소 Disk는 20Gi입니다. CPU, Memory, OS Image를 입력하지 않으면 기본값 CPU 2, Memory 4Gi, ubuntu-22.04가 적용됩니다.'}
          </div>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Button onClick={() => setIsModalVisible(false)} style={{ marginRight: 8 }}>Cancel</Button>
            <Button type="primary" htmlType="submit" disabled={!canCreateVm}>Create</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={<Text strong style={{ fontSize: '18px' }}>SSH 접속 가이드: {selectedVm?.name}</Text>}
        placement="right"
        width={600}
        onClose={() => setIsDrawerVisible(false)}
        open={isDrawerVisible}
      >
        {selectedVm && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
            <Alert
              message="SSH 접속 안내"
              description="VM 생성 시 입력한 SSH User ID와 SSH Password로 서버에 접속하면 kite-host-agent가 해당 VM SSH 서비스로 연결합니다."
              type="info"
              showIcon
            />

            <div>
              <Title level={5}>방법 1. 한 줄 명령어로 바로 접속하기 (가장 간단함)</Title>
              <Paragraph>Kite가 설치된 서버 주소로 접속합니다.</Paragraph>
              <pre style={{ backgroundColor: '#f4f4f4', padding: '16px', borderRadius: '4px', overflowX: 'auto', border: '1px solid #d9d9d9' }}>
                <code style={{ fontFamily: 'monospace' }}>ssh {selectedVm.sshId}@{MOCK_ENV.BASE_DOMAIN}</code>
              </pre>
            </div>

            <div>
              <Title level={5}>방법 2. ~/.ssh/config 수정하고 편하게 들어가기</Title>
              <Paragraph>매번 길게 치기 귀찮다면, 아래 내용을 <Text keyboard>~/.ssh/config</Text>에 추가하세요. (초기 1회 설정)</Paragraph>
              <pre style={{ backgroundColor: '#f4f4f4', padding: '16px', borderRadius: '4px', overflowX: 'auto', border: '1px solid #d9d9d9' }}>
                <code style={{ fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
{`Host bastion
    HostName ${MOCK_ENV.BASE_DOMAIN}
    User ${selectedVm.sshId}
    Port 22
`}
                </code>
              </pre>
              <Paragraph>이후에는 터미널에서 <Text keyboard>ssh bastion</Text> 만 입력하면 해당 VM으로 접속됩니다.</Paragraph>
            </div>
          </div>
        )}
      </Drawer>
    </Layout>
  );
};
