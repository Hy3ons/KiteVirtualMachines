import { useEffect, useState } from 'react';
import { App } from 'antd';
import { adminApi } from '../../api';
import type { ConfigResponse } from '../../api/types';
import { useRoutingSettings } from './useRoutingSettings';
import { useGatewaySettings } from './useGatewaySettings';
import { useSecuritySettings } from './useSecuritySettings';

// Loads configuration once and composes independent settings controllers.
export function useAdminSettings(initialData: ConfigResponse | null = null) {
  const { message } = App.useApp();
  const [data, setData] = useState(initialData);
  useEffect(() => {
    if (initialData) return;
    let active = true;
    void adminApi.getSettings().then((result) => {
      if (active) setData(result);
    }).catch((error: unknown) => {
      if (!(error instanceof Error)) throw error;
      if (active) message.error('시스템 설정을 불러오지 못했습니다. 다시 시도해 주세요.');
    });
    return () => { active = false; };
  }, [initialData, message]);
  return { ...useRoutingSettings(data), ...useGatewaySettings(data), ...useSecuritySettings(data) };
}

export type AdminSettingsState = ReturnType<typeof useAdminSettings>;
