import { useEffect, useState } from 'react';
import { App as AntdApp, Form } from 'antd';
import { adminApi } from '../../api';
import type { ConfigResponse } from '../../api/types';
import type { SSHGatewaySettings } from '../../api/types';
import { validTCPPort } from '../../domain/network';

type SSHGatewaySettingsForm = {
  readonly externalEnabled: boolean;
  readonly externalPort: string;
  readonly publicPort: string;
};

// Owns GatewaySettings forms and feedback; initialData is validated platform configuration.
export function useGatewaySettings(initialData: ConfigResponse | null) {
  const { message } = AntdApp.useApp();
  const [loadingSSHGateway, setLoadingSSHGateway] = useState(false);
  const [savedSSHGateway, setSSHGateway] = useState<SSHGatewaySettings | null>(null);
  const sshGateway = savedSSHGateway ?? initialData?.config.sshGateway ?? null;
  const [sshGatewayForm] = Form.useForm<SSHGatewaySettingsForm>();
  useEffect(() => {
    if (!initialData) return;
    const gateway = initialData.config.sshGateway;
    sshGatewayForm.setFieldsValue({ externalEnabled: gateway.externalEnabled, externalPort: gateway.externalPort, publicPort: gateway.publicPort || gateway.externalPort });
  }, [initialData, sshGatewayForm]);
  // Validates values and saves gateway exposure settings; true lets the editor close.
  // Validation or API failures return false so the modal retains the operator's input.
  const handleSaveSSHGateway = async (values: SSHGatewaySettingsForm) => {
    const externalPort = values.externalPort.trim();
    const publicPort = values.publicPort.trim() || externalPort;
    if (values.externalEnabled && !validTCPPort(externalPort)) {
      message.error('Gateway Service 포트는 1-65535 사이의 TCP 포트여야 합니다.');
      return false;
    }
    if (values.externalEnabled && !validTCPPort(publicPort)) {
      message.error('사용자 안내 포트는 1-65535 사이의 TCP 포트여야 합니다.');
      return false;
    }

    try {
      setLoadingSSHGateway(true);
      const data = await adminApi.saveSSHGateway({
        externalEnabled: values.externalEnabled,
        externalPort,
        publicPort,
      });
      setSSHGateway(data.config.sshGateway);
      message.success('SSH gateway 설정이 저장되었습니다. 컨트롤러가 외부 노출 상태를 조정합니다.');
      return true;
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error('SSH gateway 설정 저장에 실패했습니다.');
      return false;
    } finally {
      setLoadingSSHGateway(false);
    }
  };

  return { loadingSSHGateway, sshGateway, sshGatewayForm, handleSaveSSHGateway };
}
