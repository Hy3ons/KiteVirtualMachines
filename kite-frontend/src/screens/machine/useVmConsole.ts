import { useCallback, useEffect, useState } from 'react';
import { App } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { vmApi } from '../../api';
import type { UserVm } from '../../api/types';

// Loads console metadata independently from the terminal connection lifecycle.
export function useVmConsole(initialVm: UserVm | null) {
  const { message } = App.useApp();
  const { vmName } = useParams<{ vmName: string }>();
  const router = useRouter();
  const [vm, setVm] = useState<UserVm | null>(initialVm ?? null);
  const [loading, setLoading] = useState(!initialVm);

  // fetchVm refreshes the selected console's metadata without replacing the terminal connection.
  // Errors return to the VM inventory instead of opening a console for an unknown VM.
  const fetchVm = useCallback(async () => {
    try { setLoading(true); const data = await vmApi.getVm(vmName); setVm(data.vm); }
    catch (error: unknown) { if (!(error instanceof Error)) throw error; message.error(error instanceof Error ? error.message : 'VM 콘솔 정보를 불러오지 못했습니다.'); router.push('/dashboard'); }
    finally { setLoading(false); }
  }, [message, router, vmName]);
  useEffect(() => { if (!initialVm) void Promise.resolve().then(fetchVm); }, [fetchVm, initialVm]);

  return { vm, vmName, router, loading, fetchVm };
}
