'use client';

import { useState } from 'react';
import { Button, Empty, Table } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { ConsoleShell } from '../components/ConsoleShell';
import { UserDashboardSummary } from './UserDashboardSummary';
import { VmCreateModal } from './VmCreateModal';
import { VmConnectionDrawer } from './VmConnectionDrawer';
import { UserVmOffersSection } from './UserVmOffersSection';
import { UserDashboardToolbar } from './UserDashboardToolbar';
import { createUserDashboardColumns } from './UserDashboardColumns';
import { useDashboard } from './dashboard/useDashboard';
import { LEVEL_1_FIXED_CPU, LEVEL_1_FIXED_DISK_GI, LEVEL_1_FIXED_MEMORY, LEVEL_1_VM_QUOTA, MIN_DISK_GI, getAccessLevelDescription } from './userDashboardAccess';
import type { DashboardData } from '../api/pageData';

// UserDashboard presents SSR resources and client filters without changing VM API or permission contracts.
export function UserDashboard({ initialData = null }: { readonly initialData?: DashboardData | null }) {
  const dashboard = useDashboard(initialData);
  const router = useRouter();
  const [query, setQuery] = useState('');
  const [phase, setPhase] = useState('all');
  const { vms, safeAccessLevel, canCreateVm, loading, adminContact } = dashboard;
  const normalizedQuery = query.trim().toLowerCase();
  const visibleVms = vms.filter((vm) =>
    (phase === 'all' || vm.phase === phase) &&
    [vm.name, vm.domain, vm.sshId].some((value) => value.toLowerCase().includes(normalizedQuery)),
  );
  const isFiltered = normalizedQuery !== '' || phase !== 'all';
  const columns = createUserDashboardColumns({
    onNavigateToDetail: (name) => router.push(`/dashboard/kite-machine/${name}`),
    onConnect: dashboard.showConnectGuide,
    onOpenConsole: (name) => router.push(`/dashboard/kite-machine/${name}/console`),
    onStart: dashboard.handleStart, onStop: dashboard.handleStop, onDelete: dashboard.handleDelete,
    canMutateVm: dashboard.canMutateVm,
  });

  return (
    <ConsoleShell section="workspace">
      <div className="workspace-dashboard">
        <UserDashboardSummary
          accessLevel={safeAccessLevel}
          vms={vms} quotaLimit={LEVEL_1_VM_QUOTA} loading={loading}
          accessDescription={getAccessLevelDescription(safeAccessLevel)}
        />
        <section id="instances" className="workspace-resource-panel" aria-labelledby="instances-title">
          <UserDashboardToolbar
            canCreateVm={canCreateVm} loading={loading} accessLevel={safeAccessLevel} adminContact={adminContact}
            onRefresh={() => { void dashboard.fetchVms(); void dashboard.fetchOffers(); }}
            onCreate={() => dashboard.setIsModalVisible(true)} query={query} onQueryChange={setQuery}
            phase={phase} onPhaseChange={setPhase} phases={Array.from(new Set(vms.map((vm) => vm.phase)))}
          />
          <Table
            columns={columns} dataSource={visibleVms} rowKey="id" loading={loading}
            pagination={false} scroll={{ x: 800 }} className="workspace-vm-table"
            locale={{ emptyText: (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={isFiltered ? '검색 조건에 맞는 가상머신이 없습니다.' : safeAccessLevel < 1
                  ? `VM 생성 권한이 없습니다.${adminContact ? ` 관리자 연락처: ${adminContact}` : ''}`
                  : '첫 가상머신으로 워크스페이스를 시작하세요.'}>
                {isFiltered ? <Button onClick={() => { setQuery(''); setPhase('all'); }}>필터 초기화</Button> : canCreateVm && (
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => dashboard.setIsModalVisible(true)}>가상머신 만들기</Button>
                )}
              </Empty>
            ) }}
          />
        </section>
        <UserVmOffersSection offers={dashboard.offers} loading={dashboard.loadingOffers} onClaim={dashboard.handleClaimOffer} />
      </div>
      <VmCreateModal
        open={dashboard.isModalVisible} onCancel={() => dashboard.setIsModalVisible(false)}
        onCreate={dashboard.handleCreate} form={dashboard.form} canCreateVm={canCreateVm}
        accessLevel={safeAccessLevel} fixedCpu={LEVEL_1_FIXED_CPU} fixedMemory={LEVEL_1_FIXED_MEMORY}
        fixedDiskGi={LEVEL_1_FIXED_DISK_GI} minDiskGi={MIN_DISK_GI} adminContact={adminContact} baseDomain={dashboard.baseDomain}
      />
      <VmConnectionDrawer
        open={dashboard.isDrawerVisible} vm={dashboard.selectedVm} baseDomain={dashboard.baseDomain}
        sshGateway={dashboard.sshGateway} onClose={() => dashboard.setIsDrawerVisible(false)}
      />
    </ConsoleShell>
  );
}
