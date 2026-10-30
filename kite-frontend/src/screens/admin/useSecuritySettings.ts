import { useState } from 'react';
import { App as AntdApp, Form } from 'antd';
import { adminApi } from '../../api';
import type { ConfigResponse, RuntimeSecretRotation } from '../../api/types';

type CertificateSettingsForm = {
  readonly tlsCert: string;
  readonly tlsKey: string;
};

// Owns SecuritySettings forms and feedback; initialData is validated platform configuration.
export function useSecuritySettings(initialData: ConfigResponse | null) {
  const { message } = AntdApp.useApp();
  const [loadingRuntime, setLoadingRuntime] = useState(false);
  const [loadingCert, setLoadingCert] = useState(false);
  const [savedRuntimeStatus, setRuntimeStatus] = useState<{ readonly hasJWTSecret: boolean; readonly hasPasswordSalt: boolean } | null>(null);
  const runtimeStatus = savedRuntimeStatus ?? (initialData ? { hasJWTSecret: initialData.config.hasJWTSecret, hasPasswordSalt: initialData.config.hasPasswordSalt } : null);
  const [certForm] = Form.useForm<CertificateSettingsForm>();
  // Rotates only the secrets selected by payload and updates their presence indicators.
  const handleRotateRuntimeSecrets = async (payload: RuntimeSecretRotation) => {
    try {
      setLoadingRuntime(true);
      const data = await adminApi.rotateRuntimeSecrets(payload);
      setRuntimeStatus({
        hasJWTSecret: data.config.hasJWTSecret,
        hasPasswordSalt: data.config.hasPasswordSalt,
      });
      message.success('런타임 secret이 새 값으로 저장되었습니다. 이 값은 kite-api 재시작 후 적용됩니다.');
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error('런타임 secret 회전에 실패했습니다.');
    } finally {
      setLoadingRuntime(false);
    }
  };

  // Saves the supplied PEM certificate and key, then clears the sensitive form fields.
  const handleSaveCert = async (values: CertificateSettingsForm) => {
    try {
      setLoadingCert(true);
      await adminApi.saveCert({ tlsCert: values.tlsCert, tlsKey: values.tlsKey });
      message.success('HTTPS 인증서가 kite/global-tls-secret에 저장/갱신되었습니다.');
      certForm.resetFields();
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error('인증서 저장에 실패했습니다.');
    } finally {
      setLoadingCert(false);
    }
  };

  return { loadingRuntime, loadingCert, runtimeStatus, certForm, handleRotateRuntimeSecrets, handleSaveCert };
}
