'use client';

import { useState } from 'react';
import { Typography, Table, Space, Button, Popconfirm, Tabs, Select, Input } from 'antd';
import type { TableColumnsType } from 'antd';
import { PoweroffOutlined, DeleteOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { ConsoleShell } from '../components/ConsoleShell';
import { StatusBadge } from '../components/StatusBadge';
import { AdminVmOfferPanel } from './AdminVmOfferPanel';
import { useAdminDashboard } from './admin/useAdminDashboard';
import type { AdminDashboardData } from '../api/pageData';
import type { AccessLevel, AdminUser, GlobalVm } from '../api/types';

const { Text } = Typography;
const accessOptions = [
  { value: 0, label: 'Level 0 · 배정형 사용자' },
  { value: 1, label: 'Level 1 · 사용자' },
  { value: 2, label: 'Level 2 · 매니저' },
  { value: 3, label: 'Level 3 · 관리자' },
] as const;

// AdminDashboard presents role-scoped resources and actions; initialData is verified server data.
export function AdminDashboard({ initialData = null }: { readonly initialData?: AdminDashboardData | null }) {
  const { canManageAll, users, vms, loadingUsers, loadingVms, creatingOffer, fetchAllData, handleChangeAccessLevel, handleDeleteUser, handleForceStopVm, handleDeleteVm, handleCreateOffer } = useAdminDashboard(initialData);
  const [userQuery, setUserQuery] = useState('');
  const [vmQuery, setVmQuery] = useState('');
  const filteredUsers = users.filter((user) => `${user.username} ${user.email} ${user.namespace}`.toLowerCase().includes(userQuery.toLowerCase()));
  const filteredVms = vms.filter((vm) => `${vm.name} ${vm.owner} ${vm.domain} ${vm.namespace}`.toLowerCase().includes(vmQuery.toLowerCase()));
  const userColumns: TableColumnsType<AdminUser> = [
    { title: '사용자', dataIndex: 'username', key: 'username', render: (t: string) => <Text strong>{t}</Text> },
    { title: '이메일', dataIndex: 'email', key: 'email' },
    { title: '네임스페이스', dataIndex: 'namespace', key: 'namespace', render: (t: string) => <Text code>{t}</Text> },
    { title: '상태', dataIndex: 'status', key: 'status', render: (status: string) => (
      <StatusBadge phase={status} />
    )},
    {
      title: '권한',
      key: 'accessLevel',
      width: 250,
      render: (_, record) => (
        <Select
          className="admin-access-select"
          aria-label={`${record.username} 사용자 권한`}
          value={record.accessLevel}
          options={accessOptions.filter((option) => canManageAll || option.value <= 1 || option.value === record.accessLevel)}
          onChange={(value: AccessLevel) => handleChangeAccessLevel(record.username, value)}
          disabled={record.username === 'admin' || (!canManageAll && record.accessLevel > 1)}
        />
      )
    },
    {
      title: '작업',
      key: 'actions',
      render: (_, record) => (
        canManageAll ? (
          <Popconfirm title="정말 유저와 네임스페이스 전체를 삭제하시겠습니까?" onConfirm={() => handleDeleteUser(record.username)} disabled={record.username === 'admin'}>
            <Button type="text" danger icon={<DeleteOutlined />} disabled={record.username === 'admin'}>삭제</Button>
          </Popconfirm>
        ) : (
          <Text type="secondary">권한 0/1 변경만 가능</Text>
        )
      )
    }
  ];

  const vmColumns: TableColumnsType<GlobalVm> = [
    { title: '소유자', dataIndex: 'owner', key: 'owner', render: (t: string) => <Text strong>{t}</Text> },
    { title: 'VM 이름', dataIndex: 'name', key: 'name', render: (t: string) => <Text strong>{t}</Text> },
    { title: '도메인', dataIndex: 'domain', key: 'domain' },
    { title: '상태', dataIndex: 'phase', key: 'phase', render: (phase: string) => <StatusBadge phase={phase} /> },
    { title: 'CPU / 메모리', key: 'spec', render: (_, record) => `${record.cpu} / ${record.memory}` },
    {
      title: '관리 작업',
      key: 'actions',
      render: (_, record) => (
        <Space size="middle">
          {record.phase === 'Running' && (
            <Popconfirm title="관리자 권한으로 강제 종료하시겠습니까?" onConfirm={() => handleForceStopVm(record.namespace, record.name)}>
              <Button type="text" danger icon={<PoweroffOutlined />}>강제 종료</Button>
            </Popconfirm>
          )}
          <Popconfirm title="관리자 권한으로 완전 삭제하시겠습니까?" onConfirm={() => handleDeleteVm(record.namespace, record.name)}>
            <Button type="text" danger icon={<DeleteOutlined />}>강제 삭제</Button>
          </Popconfirm>
        </Space>
      )
    }
  ];

  return (
    <ConsoleShell section="admin" eyebrow="플랫폼 관리" title="관리자 콘솔"
      actions={<Button icon={<ReloadOutlined />} loading={loadingUsers || loadingVms} onClick={() => void fetchAllData()}>새로고침</Button>}>
      <section className="admin-resource-panel" aria-label="관리 리소스">
        <Tabs defaultActiveKey="users" className="admin-resource-tabs" items={[
          { key: 'users', label: `사용자 (${users.length})`, children: <>
            <div className="admin-table-toolbar">
              <div><h2>사용자 관리</h2></div>
              <Input className="admin-search" prefix={<SearchOutlined />} aria-label="사용자 검색" placeholder="사용자, 이메일, 네임스페이스 검색" allowClear value={userQuery} onChange={(event) => setUserQuery(event.target.value)} />
            </div>
            <Table columns={userColumns} dataSource={filteredUsers} rowKey="username" loading={loadingUsers} pagination={false} scroll={{ x: 'max-content' }} locale={{ emptyText: userQuery ? '검색 결과가 없습니다.' : '등록된 사용자가 없습니다.' }} />
          </> },
          ...(canManageAll ? [
            { key: 'vms', label: `가상머신 (${vms.length})`, children: <>
              <div className="admin-table-toolbar">
                <div><h2>가상머신 관리</h2></div>
                <Input className="admin-search" prefix={<SearchOutlined />} aria-label="가상머신 검색" placeholder="이름, 소유자, 도메인 검색" allowClear value={vmQuery} onChange={(event) => setVmQuery(event.target.value)} />
              </div>
              <Table columns={vmColumns} dataSource={filteredVms} rowKey="id" loading={loadingVms} pagination={false} scroll={{ x: 'max-content' }} locale={{ emptyText: vmQuery ? '검색 결과가 없습니다.' : '등록된 가상머신이 없습니다.' }} />
            </> },
            { key: 'offers', label: 'VM 사전 할당', children: <AdminVmOfferPanel users={users} loading={creatingOffer} onCreate={handleCreateOffer} /> },
          ] : []),
        ]} />
      </section>
    </ConsoleShell>
  );
}
