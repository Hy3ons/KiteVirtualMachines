import { canManageSystem } from '../../domain/access';
import { useState, useEffect, useCallback } from 'react';
import { App as AntdApp } from 'antd';
import { adminApi } from '../../api';
import { useAuthStore } from '../../store/useAuthStore';
import type { AdminDashboardData } from '../../api/pageData';
import type { AccessLevel, AdminCreateVmOfferPayload, AdminUser, GlobalVm } from '../../api/types';

// useAdminDashboard retains role-scoped data and administrator actions using verified SSR initialData.
export function useAdminDashboard(initialData: AdminDashboardData | null = null) {
  const { message } = AntdApp.useApp();
  const { accessLevel } = useAuthStore();
  const canManageAll = canManageSystem(accessLevel);

  const [users, setUsers] = useState<readonly AdminUser[]>(initialData?.users ?? []);
  const [vms, setVms] = useState<readonly GlobalVm[]>(initialData?.vms ?? []);
  const [loadingUsers, setLoadingUsers] = useState(!initialData);
  const [loadingVms, setLoadingVms] = useState(canManageAll && !initialData);
  const [creatingOffer, setCreatingOffer] = useState(false);

  // Refreshes users and, only for full administrators, all VM resources.
  const fetchAllData = useCallback(async () => {
    try {
      setLoadingUsers(true);
      if (canManageAll) {
        setLoadingVms(true);
      }
      const usersRes = await adminApi.getUsers();
      setUsers(usersRes.users);
      if (canManageAll) {
        const vmsRes = await adminApi.getVms();
        setVms(vmsRes.vms);
      } else {
        setVms([]);
      }
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error('관리자 데이터를 불러오는데 실패했습니다.');
    } finally {
      setLoadingUsers(false);
      if (canManageAll) {
        setLoadingVms(false);
      }
    }
  }, [canManageAll, message]);

  useEffect(() => {
    if (!initialData) void Promise.resolve().then(fetchAllData);
  }, [fetchAllData, initialData]);

  // Updates targetUser to newLevel selected within the permitted role bounds.
  const handleChangeAccessLevel = async (targetUser: string, newLevel: AccessLevel) => {
    try {
      message.loading({ content: '권한을 변경하고 있습니다…', key: 'level' });
      await adminApi.updateUserAccess(targetUser, newLevel);
      message.success({ content: '권한이 업데이트 되었습니다.', key: 'level' });
      fetchAllData();
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error({ content: '권한 업데이트 실패', key: 'level' });
    }
  };

  // Deletes targetUser and their namespace after the administrator confirms.
  const handleDeleteUser = async (targetUser: string) => {
    try {
      message.loading({ content: '사용자를 삭제하고 있습니다…', key: 'delete-user' });
      await adminApi.deleteUser(targetUser);
      message.success({ content: '유저 및 네임스페이스가 삭제되었습니다.', key: 'delete-user' });
      fetchAllData();
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error({ content: '유저 삭제 실패', key: 'delete-user' });
    }
  };

  // Requests a force stop for the named VM in namespace after confirmation.
  const handleForceStopVm = async (namespace: string, name: string) => {
    try {
      message.loading({ content: 'VM 종료를 요청하고 있습니다…', key: 'stop-vm' });
      await adminApi.forceStopVm(namespace, name);
      message.success({ content: '가상머신이 강제 종료 요청되었습니다.', key: 'stop-vm' });
      fetchAllData();
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error({ content: '종료 요청 실패', key: 'stop-vm' });
    }
  };

  // Deletes the named VM in namespace after administrator confirmation.
  const handleDeleteVm = async (namespace: string, name: string) => {
    try {
      message.loading({ content: 'VM 삭제를 요청하고 있습니다…', key: 'delete-vm' });
      await adminApi.deleteVm(namespace, name);
      message.success({ content: '가상머신이 강제 삭제 요청되었습니다.', key: 'delete-vm' });
      fetchAllData();
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error({ content: '삭제 요청 실패', key: 'delete-vm' });
    }
  };

  // Creates the supplied offer allocation; false preserves the form after a failed request.
  const handleCreateOffer = async (payload: AdminCreateVmOfferPayload) => {
    try {
      setCreatingOffer(true);
      await adminApi.createVmOffer(payload);
      message.success('VM offer가 생성되었습니다. 대상 사용자 대시보드에 표시됩니다.');
      return true;
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error('VM offer 생성에 실패했습니다.');
      return false;
    } finally {
      setCreatingOffer(false);
    }
  };

  return { canManageAll, users, vms, loadingUsers, loadingVms, creatingOffer, fetchAllData, handleChangeAccessLevel, handleDeleteUser, handleForceStopVm, handleDeleteVm, handleCreateOffer };
}
