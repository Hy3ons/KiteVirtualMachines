import { useEffect, useState } from 'react';
import { App as AntdApp, Form } from 'antd';
import { adminApi } from '../../api';
import type { ConfigResponse } from '../../api/types';

type DomainSettingsForm = {
  readonly baseDomain: string;
};

type AdminContactForm = {
  readonly adminContact: string;
};

// Owns RoutingSettings forms and feedback; initialData is validated platform configuration.
export function useRoutingSettings(initialData: ConfigResponse | null) {
  const { message } = AntdApp.useApp();
  const [loadingDomain, setLoadingDomain] = useState(false);
  const [loadingHTTPS, setLoadingHTTPS] = useState(false);
  const [loadingContact, setLoadingContact] = useState(false);
  const [savedForceHTTPS, setForceHTTPS] = useState<boolean | null>(null);
  const forceHTTPS = savedForceHTTPS ?? initialData?.config.forceHttps ?? false;
  const [domainForm] = Form.useForm<DomainSettingsForm>();
  const [contactForm] = Form.useForm<AdminContactForm>();
  useEffect(() => {
    if (!initialData) return;
    domainForm.setFieldsValue({ baseDomain: initialData.config.baseDomain });
    contactForm.setFieldsValue({ adminContact: initialData.config.adminContact });
  }, [initialData, domainForm, contactForm]);
  // Saves the supplied base domain through the settings API for the routing form.
  const handleSaveDomain = async (values: DomainSettingsForm) => {
    try {
      setLoadingDomain(true);
      await adminApi.saveDomain(values.baseDomain);
      message.success('베이스 도메인이 저장되었습니다. 컨트롤러가 기존 VM들의 Ingress를 재조정(Reconcile)합니다.');
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
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
      setForceHTTPS(data.config.forceHttps);
      message.success(checked ? '플랫폼 HTTPS 강제가 활성화되었습니다.' : '플랫폼 HTTPS 강제가 비활성화되었습니다.');
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
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
    } catch (error: unknown) {
      if (!(error instanceof Error)) throw error;
      message.error('관리자 연락처 저장에 실패했습니다.');
    } finally {
      setLoadingContact(false);
    }
  };

  return { loadingDomain, loadingHTTPS, loadingContact, forceHTTPS, domainForm, contactForm, handleSaveDomain, handleToggleForceHTTPS, handleSaveAdminContact };
}
