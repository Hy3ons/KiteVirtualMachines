'use client';

import { useEffect, useState } from 'react';
import { vmApi } from '../api';
import { accountVmUsage } from '../api/accountUsage';
import type { AccountVmUsage as VmUsage } from '../api/accountUsage';

type UsageState = { readonly status: 'loading' } | { readonly status: 'loaded'; readonly usage: VmUsage } | { readonly status: 'error' };

// AccountVmUsage is mounted only in an open account popover and refreshes the existing /vms data.
// Closing the panel invalidates in-flight results, so another account cannot inherit stale counts.
export function AccountVmUsage() {
  const [state, setState] = useState<UsageState>({ status: 'loading' });
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    vmApi.getVms().then((response) => {
      const usage = accountVmUsage(response.vms);
      if (active) setState({ status: 'loaded', usage });
    }).catch((error: unknown) => {
      if (!(error instanceof Error)) throw error;
      if (active) setState({ status: 'error' });
    });
    return () => { active = false; };
  }, [attempt]);

  switch (state.status) {
    case 'loading':
      return <div className="account-usage-state" role="status">VM 정보를 불러오는 중…</div>;
    case 'error':
      return <div className="account-usage-state" role="status"><span>VM 정보를 불러오지 못했습니다.</span><button type="button" className="account-retry appbar-action" onClick={() => { setState({ status: 'loading' }); setAttempt((value) => value + 1); }}>다시 시도</button></div>;
    case 'loaded':
      return <dl className="account-vm-usage" aria-label="내 VM 이용 현황"><div><dt>보유 가상머신</dt><dd>{state.usage.total}<small>개</small></dd></div><div><dt>실행 중</dt><dd>{state.usage.running}<small>개</small></dd></div></dl>;
    default: {
      const unexpected: never = state;
      throw new Error(`Unknown account usage state: ${String(unexpected)}`);
    }
  }
}
