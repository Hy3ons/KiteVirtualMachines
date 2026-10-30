import { useEffect } from 'react';
import { Empty, Form, Table } from 'antd';
import { UserDashboardSummary } from '../../../screens/UserDashboardSummary';
import { UserDashboardToolbar } from '../../../screens/UserDashboardToolbar';
import { createUserDashboardColumns } from '../../../screens/UserDashboardColumns';
import { UserVmOffersSection } from '../../../screens/UserVmOffersSection';
import { VmCreateModal } from '../../../screens/VmCreateModal';
import type { VmCreateFormValues } from '../../../screens/UserDashboardTypes';
import { getAccessLevelDescription } from '../../../screens/userDashboardAccess';
import type { DemoSnapshot } from './demoStory';
import { DEMO_VM, demoIgnore, demoIgnoreAsync } from './demoData';
import { DemoDeleteModal } from './DemoDeleteModal';

// DemoDashboard composes the actual workspace widgets with explicitly local sample data.
// Modal and row menu are controlled by the story; callbacks never call Go or change auth.
export function DemoDashboard({ snapshot }: { readonly snapshot: DemoSnapshot }) {
  const [form] = Form.useForm<VmCreateFormValues>();
  useEffect(() => {
    if (snapshot.scene === 'create') form.setFieldsValue({ name: snapshot.name, domainPrefix: snapshot.domainPrefix, sshId: snapshot.sshId, sshPassword: snapshot.sshPassword, disk: 25 });
  }, [form, snapshot.scene, snapshot.name, snapshot.domainPrefix, snapshot.sshId, snapshot.sshPassword]);
  const vms = snapshot.hasVm ? [{ ...DEMO_VM, phase: snapshot.phase, delete: snapshot.deleting }] : [];
  const columns = createUserDashboardColumns({ onNavigateToDetail: demoIgnore, onConnect: demoIgnore, onOpenConsole: demoIgnore, onStart: demoIgnoreAsync, onStop: demoIgnoreAsync, onDelete: demoIgnoreAsync, canMutateVm: true, previewMenuOpen: snapshot.menuOpen });
  return <div className="demo-dashboard">
    <div className="workspace-dashboard">
      <UserDashboardSummary accessLevel={1} vms={vms} quotaLimit={2} loading={false} accessDescription={getAccessLevelDescription(1)} />
      <section className="workspace-resource-panel" aria-labelledby="instances-title">
        <UserDashboardToolbar canCreateVm loading={false} accessLevel={1} adminContact="" onRefresh={demoIgnore} onCreate={demoIgnore} query="" onQueryChange={demoIgnore} phase="all" onPhaseChange={demoIgnore} phases={vms.map((vm) => vm.phase)} />
        <Table columns={columns} dataSource={vms} rowKey="id" pagination={false} scroll={{ x: 800 }} className="workspace-vm-table" locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="첫 가상머신으로 워크스페이스를 시작하세요." /> }} />
      </section>
      <UserVmOffersSection offers={[]} loading={false} onClaim={async () => false} />
    </div>
    <VmCreateModal open={snapshot.scene === 'create'} onCancel={demoIgnore} onCreate={demoIgnoreAsync} form={form} canCreateVm accessLevel={1} fixedCpu={2} fixedMemory="4Gi" fixedDiskGi={25} minDiskGi={20} adminContact="" baseDomain="apps.example.com" formName="demo_vm_create" getContainer={false} focusable={{ trap: false, focusTriggerAfterClose: false }} submitting={snapshot.busy} />
    <DemoDeleteModal snapshot={snapshot} />
  </div>;
}
