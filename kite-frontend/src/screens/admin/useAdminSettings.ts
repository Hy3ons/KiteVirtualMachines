import { useEffect, useState } from 'react';
import { App as AntdApp, Form } from 'antd';
import { adminApi } from '../../api';
import type { ConfigResponse, SSHGatewaySettings } from '../../api/types';

type DomainSettingsForm = {
  readonly baseDomain: string;
};

type CertificateSettingsForm = {
  readonly tlsCert: string;
  readonly tlsKey: string;
};

type AdminContactForm = {
  readonly adminContact: string;
};

type SSHGatewaySettingsForm = {
  readonly externalEnabled: boolean;
  readonly externalPort: string;
  readonly publicPort: string;
};

// validPort accepts a form value and checks the TCP port range before gateway saves.
const validPort = (value: string): boolean => {
  const port = Number(value.trim());
  return Number.isInteger(port) && port >= 1 && port <= 65535;
};

// useAdminSettings owns the settings forms and save operations; initialData is verified SSR configuration.
export function useAdminSettings(initialData?: ConfigResponse) {
  const { message } = AntdApp.useApp();
  const [loadingDomain, setLoadingDomain] = useState(false);
  const [loadingHTTPS, setLoadingHTTPS] = useState(false);
  const [loadingContact, setLoadingContact] = useState(false);
  const [loadingRuntime, setLoadingRuntime] = useState(false);
  const [loadingCert, setLoadingCert] = useState(false);
  const [loadingSSHGateway, setLoadingSSHGateway] = useState(false);
  const [forceHTTPS, setForceHTTPS] = useState(initialData?.config.forceHttps ?? false);
  const [runtimeStatus, setRuntimeStatus] = useState<{ hasJWTSecret?: boolean; hasPasswordSalt?: boolean }>({ hasJWTSecret: initialData?.config.hasJWTSecret, hasPasswordSalt: initialData?.config.hasPasswordSalt });
  const [sshGateway, setSSHGateway] = useState<SSHGatewaySettings | undefined>(initialData?.config.sshGateway);
  
  const [domainForm] = Form.useForm<DomainSettingsForm>();
  const [contactForm] = Form.useForm<AdminContactForm>();
  const [sshGatewayForm] = Form.useForm<SSHGatewaySettingsForm>();
  const [certForm] = Form.useForm<CertificateSettingsForm>();

  useEffect(() => {
    // 최초에 설정 불러오기
    (initialData ? Promise.resolve(initialData) : adminApi.getSettings()).then(data => {
      if (data.config) {
        domainForm.setFieldsValue({ baseDomain: data.config.baseDomain });
        contactForm.setFieldsValue({ adminContact: data.config.adminContact });
        setForceHTTPS(Boolean(data.config.forceHttps));
        setRuntimeStatus({
          hasJWTSecret: data.config.hasJWTSecret,
          hasPasswordSalt: data.config.hasPasswordSalt,
        });
        setSSHGateway(data.config.sshGateway);
        sshGatewayForm.setFieldsValue({
          externalEnabled: Boolean(data.config.sshGateway?.externalEnabled),
          externalPort: data.config.sshGateway?.externalPort || '',
          publicPort: data.config.sshGateway?.publicPort || data.config.sshGateway?.externalPort || '',
        });
      }
    }).catch(() => {
      message.error('시스템 설정을 불러오지 못했습니다. 다시 시도해 주세요.');
    });
  }, [contactForm, domainForm, sshGatewayForm, initialData, message]);

  // Saves the supplied base domain through the settings API for the routing form.
  const handleSaveDomain = async (values: DomainSettingsForm) => {
    try {
      setLoadingDomain(true);
      await adminApi.saveDomain(values.baseDomain);
      message.success('베이스 도메인이 저장되었습니다. 컨트롤러가 기존 VM들의 Ingress를 재조정(Reconcile)합니다.');
    } catch {
      message.error('베이스 도메인 저장에 실패했습니다.');
    } finally {
      setLoadingDomain(false);
    }
  };

  // Persists the switch value and uses the confirmed server policy for the HTTPS control.
  const handleToggleForceHTTPS = async (checked: boolean) => {
    try {
      setLoadingHTTPS(true);
      const data = await adminApi.saveHTTPSPolicy({ forceHttps: checked });
      if (data.config) {
        setForceHTTPS(data.config.forceHttps);
      }
      message.success(checked ? '플랫폼 HTTPS 강제가 활성화되었습니다.' : '플랫폼 HTTPS 강제가 비활성화되었습니다.');
    } catch {
      message.error('HTTPS 강제 설정 변경에 실패했습니다.');
    } finally {
      setLoadingHTTPS(false);
    }
  };

  // Saves the supplied operator contact shown to users without VM creation access.
  const handleSaveAdminContact = async (values: AdminContactForm) => {
    try {
      setLoadingContact(true);
      await adminApi.saveAdminContact(values.adminContact);
      message.success('관리자 연락처가 저장되었습니다. Level 0 사용자 안내에 반영됩니다.');
    } catch {
      message.error('관리자 연락처 저장에 실패했습니다.');
    } finally {
      setLoadingContact(false);
    }
  };

  // Validates values and saves gateway exposure settings; true lets the editor close.
  // Validation or API failures return false so the modal retains the operator's input.
  const handleSaveSSHGateway = async (values: SSHGatewaySettingsForm) => {
    const externalPort = values.externalPort?.trim() || '';
    const publicPort = values.publicPort?.trim() || externalPort;
    if (values.externalEnabled && !validPort(externalPort)) {
      message.error('Gateway Service 포트는 1-65535 사이의 TCP 포트여야 합니다.');
      return false;
    }
    if (values.externalEnabled && !validPort(publicPort)) {
      message.error('사용자 안내 포트는 1-65535 사이의 TCP 포트여야 합니다.');
      return false;
    }

    try {
      setLoadingSSHGateway(true);
      const data = await adminApi.saveSSHGateway({
        externalEnabled: Boolean(values.externalEnabled),
        externalPort,
        publicPort,
      });
      if (data.config?.sshGateway) {
        setSSHGateway(data.config.sshGateway);
      }
      message.success('SSH gateway 설정이 저장되었습니다. 컨트롤러가 외부 노출 상태를 조정합니다.');
      return true;
    } catch {
      message.error('SSH gateway 설정 저장에 실패했습니다.');
      return false;
    } finally {
      setLoadingSSHGateway(false);
    }
  };

  // Rotates only the secrets selected by payload and updates their presence indicators.
  const handleRotateRuntimeSecrets = async (payload: { rotateJWTSecret?: boolean; rotatePasswordSalt?: boolean }) => {
    try {
      setLoadingRuntime(true);
      const data = await adminApi.rotateRuntimeSecrets(payload);
      if (data.config) {
        setRuntimeStatus({
          hasJWTSecret: data.config.hasJWTSecret,
          hasPasswordSalt: data.config.hasPasswordSalt,
        });
      }
      message.success('런타임 secret이 새 값으로 저장되었습니다. 이 값은 kite-api 재시작 후 적용됩니다.');
    } catch {
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
    } catch {
      message.error('인증서 저장에 실패했습니다.');
    } finally {
      setLoadingCert(false);
    }
  };

  return { loadingDomain, loadingHTTPS, loadingContact, loadingRuntime, loadingCert, loadingSSHGateway, forceHTTPS, runtimeStatus, sshGateway, domainForm, contactForm, sshGatewayForm, certForm, handleSaveDomain, handleToggleForceHTTPS, handleSaveAdminContact, handleSaveSSHGateway, handleRotateRuntimeSecrets, handleSaveCert };
}

export type AdminSettingsState = ReturnType<typeof useAdminSettings>;
