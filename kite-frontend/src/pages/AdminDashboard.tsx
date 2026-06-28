import React, { useState, useEffect, useCallback } from 'react';
import { SEO } from '../components/SEO';
import { adminApi } from '../api';
import { App as AntdApp, Layout, Typography, Table, Space, Tag, Button, Popconfirm, Tabs, Statistic, Row, Col, Card, Slider, Avatar } from 'antd';
import type { TableColumnsType } from 'antd';
import { PoweroffOutlined, DeleteOutlined, LogoutOutlined, TeamOutlined, CloudServerOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { useNavigate } from 'react-router-dom';
import { GlobalHeader } from '../components/GlobalHeader';

const { Content } = Layout;
const { Text } = Typography;

interface User {
  username: string;
  email: string;
  namespace: string;
  accessLevel: number;
  status: 'Active' | 'Pending';
}

interface GlobalVM {
  id: string;
  owner: string; // namespace
  namespace: string;
  name: string;
  domain: string;
  phase: string;
  cpu: number;
  memory: string;
}

export const AdminDashboard: React.FC = () => {
  const { message } = AntdApp.useApp();
  const { username, logout, profileImage } = useAuthStore();
  const navigate = useNavigate();
  
  const [users, setUsers] = useState<User[]>([]);
  const [vms, setVms] = useState<GlobalVM[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(true);
  const [loadingVms, setLoadingVms] = useState(true);

  const fetchAllData = useCallback(async () => {
    try {
      setLoadingUsers(true);
      setLoadingVms(true);
      const [usersRes, vmsRes] = await Promise.all([
        adminApi.getUsers(),
        adminApi.getVms()
      ]);
      setUsers(usersRes.users || []);
      setVms(vmsRes.vms || []);
    } catch {
      message.error('관리자 데이터를 불러오는데 실패했습니다.');
    } finally {
      setLoadingUsers(false);
      setLoadingVms(false);
    }
  }, [message]);

  useEffect(() => {
    void Promise.resolve().then(fetchAllData);
  }, [fetchAllData]);

  const handleChangeAccessLevel = async (targetUser: string, newLevel: number) => {
    try {
      message.loading({ content: 'Updating access level...', key: 'level' });
      await adminApi.updateUserAccess(targetUser, newLevel);
      message.success({ content: '권한이 업데이트 되었습니다.', key: 'level' });
      fetchAllData();
    } catch {
      message.error({ content: '권한 업데이트 실패', key: 'level' });
    }
  };

  const handleDeleteUser = async (targetUser: string) => {
    try {
      message.loading({ content: 'Deleting user...', key: 'delete-user' });
      await adminApi.deleteUser(targetUser);
      message.success({ content: '유저 및 네임스페이스가 삭제되었습니다.', key: 'delete-user' });
      fetchAllData();
    } catch {
      message.error({ content: '유저 삭제 실패', key: 'delete-user' });
    }
  };

  const handleForceStopVm = async (namespace: string, name: string) => {
    try {
      message.loading({ content: 'Stopping VM...', key: 'stop-vm' });
      await adminApi.forceStopVm(namespace, name);
      message.success({ content: '가상머신이 강제 종료 요청되었습니다.', key: 'stop-vm' });
      fetchAllData();
    } catch {
      message.error({ content: '종료 요청 실패', key: 'stop-vm' });
    }
  };

  const handleDeleteVm = async (namespace: string, name: string) => {
    try {
      message.loading({ content: 'Deleting VM...', key: 'delete-vm' });
      await adminApi.deleteVm(namespace, name);
      message.success({ content: '가상머신이 강제 삭제 요청되었습니다.', key: 'delete-vm' });
      fetchAllData();
    } catch {
      message.error({ content: '삭제 요청 실패', key: 'delete-vm' });
    }
  };

  const userColumns: TableColumnsType<User> = [
    { title: 'Username', dataIndex: 'username', key: 'username', render: (t: string) => <Text strong>{t}</Text> },
    { title: 'Email', dataIndex: 'email', key: 'email' },
    { title: 'Namespace', dataIndex: 'namespace', key: 'namespace', render: (t: string) => <Tag>{t}</Tag> },
    { title: 'Status', dataIndex: 'status', key: 'status', render: (status: string) => (
      <Tag color={status === 'Active' ? 'success' : 'warning'}>{status}</Tag>
    )},
    { 
      title: 'Access Level', 
      key: 'accessLevel',
      width: 250,
      render: (_, record) => (
        <Slider
          min={0}
          max={3}
          marks={{ 0: '0', 1: '1', 2: '2', 3: '3' }}
          step={null}
          value={record.accessLevel} 
          onChange={(val) => handleChangeAccessLevel(record.username, val)}
          disabled={record.username === 'admin'}
        />
      )
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record) => (
        <Popconfirm title="정말 유저와 네임스페이스 전체를 삭제하시겠습니까?" onConfirm={() => handleDeleteUser(record.username)} disabled={record.username === 'admin'}>
          <Button type="text" danger icon={<DeleteOutlined />} disabled={record.username === 'admin'}>Delete</Button>
        </Popconfirm>
      )
    }
  ];

  const vmColumns: TableColumnsType<GlobalVM> = [
    { title: 'Owner (NS)', dataIndex: 'owner', key: 'owner', render: (t: string) => <Text strong style={{ color: '#C9B59C' }}>{t}</Text> },
    { title: 'VM Name', dataIndex: 'name', key: 'name', render: (t: string) => <Text strong>{t}</Text> },
    { title: 'Domain', dataIndex: 'domain', key: 'domain' },
    { title: 'Status', dataIndex: 'phase', key: 'phase', render: (phase: string) => {
      let color = 'default';
      if (phase === 'Running') color = 'success';
      if (phase === 'Stopped') color = 'error';
      return <Tag color={color}>{phase}</Tag>;
    }},
    { title: 'CPU / Mem', key: 'spec', render: (_, record) => `${record.cpu} / ${record.memory}` },
    {
      title: 'Global Actions',
      key: 'actions',
      render: (_, record) => (
        <Space size="middle">
          {record.phase === 'Running' && (
            <Popconfirm title="관리자 권한으로 강제 종료하시겠습니까?" onConfirm={() => handleForceStopVm(record.namespace, record.name)}>
              <Button type="text" icon={<PoweroffOutlined style={{ color: '#ff4d4f' }} />}>Force Stop</Button>
            </Popconfirm>
          )}
          <Popconfirm title="관리자 권한으로 완전 삭제하시겠습니까?" onConfirm={() => handleDeleteVm(record.namespace, record.name)}>
            <Button type="text" danger icon={<DeleteOutlined />}>Force Delete</Button>
          </Popconfirm>
        </Space>
      )
    }
  ];

  return (
    <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
      <SEO title="관리자 대시보드 - Kite" url="/admin/dashboard" />
      <GlobalHeader 
        rightContent={
          <Space>
            <Button type="primary" ghost onClick={() => navigate('/dashboard')}>My VMs</Button>
            <Button type="text" onClick={() => navigate('/admin/settings')}>시스템 전역 설정</Button>
            <Avatar src={profileImage || '/default_profile.png'} />
            <Text strong style={{ marginLeft: 8 }}>{username} (Admin)</Text>
            <Button type="text" icon={<LogoutOutlined />} onClick={logout}>Logout</Button>
          </Space>
        }
      />

      <Content className="app-main app-main--wide admin-content">
        <Row gutter={[24, 24]} style={{ marginBottom: 40 }}>
          <Col xs={24} md={12}>
            <Card hoverable>
              <Statistic title="총 등록 유저 수" value={users.length} prefix={<TeamOutlined />} styles={{ content: { color: '#C9B59C' } }} />
            </Card>
          </Col>
          <Col xs={24} md={12}>
            <Card hoverable>
              <Statistic title="클러스터 전체 활성 VM 수" value={vms.length} prefix={<CloudServerOutlined />} styles={{ content: { color: '#333' } }} />
            </Card>
          </Col>
        </Row>

        <Tabs
          defaultActiveKey="1"
          size="large"
          items={[
            {
              key: '1',
              label: 'Users',
              children: (
                <Table 
                  columns={userColumns} 
                  dataSource={users} 
                  rowKey="username" 
                  loading={loadingUsers}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                />
              )
            },
            {
              key: '2',
              label: 'VM Control',
              children: (
                <Table 
                  columns={vmColumns} 
                  dataSource={vms} 
                  rowKey="id" 
                  loading={loadingVms}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                />
              )
            }
          ]}
        />
      </Content>
    </Layout>
  );
};
