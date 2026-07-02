import React, { useState, useEffect } from 'react';
import { SEO } from '../components/SEO';
import { adminApi } from '../api';
import { App as AntdApp, Layout, Typography, Form, Input, Button, Card, Space, Avatar, Switch } from 'antd';
import { GlobalOutlined, SafetyCertificateOutlined, LogoutOutlined, ReloadOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { useLogout } from '../hooks/useLogout';
import { useNavigate } from 'react-router-dom';
import { GlobalHeader } from '../components/GlobalHeader';
import { MOCK_ENV } from '../config/mockEnv';

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

const { Content } = Layout;
const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

export const AdminSettings: React.FC = () => {
  const { message } = AntdApp.useApp();
  const { username, profileImage } = useAuthStore();
  const logout = useLogout();
  const navigate = useNavigate();
  
  const [loadingDomain, setLoadingDomain] = useState(false);
  const [loadingHTTPS, setLoadingHTTPS] = useState(false);
  const [loadingContact, setLoadingContact] = useState(false);
  const [loadingRuntime, setLoadingRuntime] = useState(false);
  const [loadingCert, setLoadingCert] = useState(false);
  const [forceHTTPS, setForceHTTPS] = useState(false);
  const [runtimeStatus, setRuntimeStatus] = useState<{ hasJWTSecret?: boolean; hasPasswordSalt?: boolean }>({});
  
  const [domainForm] = Form.useForm();
  const [contactForm] = Form.useForm();
  const [certForm] = Form.useForm();

  useEffect(() => {
    // 최초에 설정 불러오기
    adminApi.getSettings().then(data => {
      if (data.config) {
        domainForm.setFieldsValue({ baseDomain: data.config.baseDomain });
        contactForm.setFieldsValue({ adminContact: data.config.adminContact });
        setForceHTTPS(Boolean(data.config.forceHttps));
        setRuntimeStatus({
          hasJWTSecret: data.config.hasJWTSecret,
          hasPasswordSalt: data.config.hasPasswordSalt,
        });
      }
    }).catch(() => {
      // ignore
    });
  }, [contactForm, domainForm]);

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

  const handleSaveCert = async (values: CertificateSettingsForm) => {
    try {
      setLoadingCert(true);
      await adminApi.saveCert({ tlsCert: values.tlsCert, tlsKey: values.tlsKey });
      message.success('HTTPS 인증서가 kube-system/global-tls-secret에 저장/갱신되었습니다.');
      certForm.resetFields();
    } catch {
      message.error('인증서 저장에 실패했습니다.');
    } finally {
      setLoadingCert(false);
    }
  };

  return (
    <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
      <SEO title="관리자 시스템 설정 - Kite" url="/admin/settings" />
      <GlobalHeader 
        rightContent={
          <Space>
            <Button type="text" onClick={() => navigate('/admin/dashboard')}>통계/VM 관리</Button>
            <Avatar src={profileImage || '/default_profile.png'} />
            <Text strong style={{ marginLeft: 8 }}>{username} (Admin)</Text>
            <Button type="text" icon={<LogoutOutlined />} onClick={logout}>Logout</Button>
          </Space>
        }
      />

      <Content className="app-main app-main--narrow admin-settings-content">
        <div className="admin-settings-heading">
          <Title level={2}>시스템 전역 설정</Title>
          <Text type="secondary">Kite 클러스터 전체에 적용되는 라우팅 및 보안 설정을 독립적으로 관리합니다.</Text>
        </div>

        <Card hoverable className="admin-settings-card">
          <Title level={4}><GlobalOutlined style={{ marginRight: 8 }} /> 베이스 도메인 설정</Title>
          <Paragraph style={{ color: '#666' }}>
            Kite 플랫폼 Ingress의 host와 가상머신별 Ingress 도메인에 사용할 기본 도메인을 설정합니다.
            <br />도메인 변경 시 컨트롤러가 platform Ingress를 갱신하고 VM Ingress는 다음 reconcile에서 새 도메인으로 맞춰집니다.
          </Paragraph>
          
          <Form form={domainForm} layout="vertical" onFinish={handleSaveDomain}>
            <Form.Item name="baseDomain" rules={[{ required: true, message: '도메인을 입력하세요.' }]}>
              <Input placeholder={MOCK_ENV.BASE_DOMAIN} size="large" />
            </Form.Item>
            <div style={{ textAlign: 'right' }}>
              <Button type="primary" htmlType="submit" loading={loadingDomain}>도메인 변경 및 전체 적용</Button>
            </div>
          </Form>
        </Card>

        <Card hoverable className="admin-settings-card">
          <Title level={4}><SafetyCertificateOutlined style={{ marginRight: 8 }} /> 플랫폼 HTTPS 강제</Title>
          <Paragraph style={{ color: '#666' }}>
            플랫폼 Ingress에 Traefik HTTPS redirect middleware를 연결합니다. 내부 80 포트 터널 QA가 필요할 때는 꺼두고,
            공개 도메인을 HTTPS로 운영할 때 켜세요.
          </Paragraph>
          <Space align="center" size={16} wrap>
            <Switch checked={forceHTTPS} loading={loadingHTTPS} onChange={handleToggleForceHTTPS} />
            <Text strong>{forceHTTPS ? 'HTTP 요청을 HTTPS로 강제 전환' : 'HTTP 접속 허용'}</Text>
          </Space>
        </Card>

        <Card hoverable className="admin-settings-card">
          <Title level={4}>관리자 연락처</Title>
          <Paragraph style={{ color: '#666' }}>
            VM 생성 권한이 없는 사용자에게 보여줄 연락 창구입니다. 이메일, 전화번호, Slack 채널, URL처럼 운영팀이 실제로 받는 값을 입력하세요.
          </Paragraph>
          <Form form={contactForm} layout="vertical" onFinish={handleSaveAdminContact}>
            <Form.Item name="adminContact" label="Contact">
              <Input placeholder="ops@example.com / Slack #kite-help / https://..." size="large" />
            </Form.Item>
            <div style={{ textAlign: 'right' }}>
              <Button type="primary" htmlType="submit" loading={loadingContact}>연락처 저장</Button>
            </div>
          </Form>
        </Card>

        <Card hoverable className="admin-settings-card">
          <Title level={4}>API 런타임 설정</Title>
          <Paragraph style={{ color: '#666' }}>
            JWT Secret과 Password Salt는 서버 최초 시작 시 자동 생성되어 Kubernetes ConfigMap에 저장됩니다. 원문은 화면에 표시하지 않습니다.
            <br />재생성된 값은 저장 후 kite-api 재시작 시 적용됩니다.
          </Paragraph>

          <Space style={{ marginBottom: 16 }}>
            <Text strong>JWT Secret:</Text>
            <Text>{runtimeStatus.hasJWTSecret ? '생성됨' : '없음'}</Text>
            <Text strong>Password Salt:</Text>
            <Text>{runtimeStatus.hasPasswordSalt ? '생성됨' : '없음'}</Text>
          </Space>

          <Space style={{ marginBottom: 16, display: 'flex', flexWrap: 'wrap' }}>
            <Button
              icon={<ReloadOutlined />}
              loading={loadingRuntime}
              onClick={() => handleRotateRuntimeSecrets({ rotateJWTSecret: true })}
            >
              JWT Secret 재생성
            </Button>
            <Button
              icon={<ReloadOutlined />}
              loading={loadingRuntime}
              onClick={() => handleRotateRuntimeSecrets({ rotatePasswordSalt: true })}
            >
              Password Salt 재생성
            </Button>
            <Button
              danger
              icon={<ReloadOutlined />}
              loading={loadingRuntime}
              onClick={() => handleRotateRuntimeSecrets({ rotateJWTSecret: true, rotatePasswordSalt: true })}
            >
              둘 다 재생성
            </Button>
          </Space>

        </Card>

        <Card hoverable className="admin-settings-card">
          <Title level={4}><SafetyCertificateOutlined style={{ marginRight: 8 }} /> 와일드카드 인증서 갱신/등록</Title>
          <Paragraph style={{ color: '#666' }}>
            등록된 도메인의 서브도메인을 모두 커버하는 TLS 인증서를 갱신합니다. 저장 시 즉시 K8s의 <Text strong>kube-system/global-tls-secret</Text>이 교체됩니다.
          </Paragraph>
          
          <Form form={certForm} layout="vertical" onFinish={handleSaveCert}>
            <Form.Item name="tlsCert" label="TLS Fullchain Certificate (.crt / .pem)" rules={[{ required: true, message: '인증서 내용을 입력해주세요.' }]}>
              <TextArea rows={6} placeholder="-----BEGIN CERTIFICATE-----..." style={{ fontFamily: 'monospace', fontSize: '13px' }} />
            </Form.Item>

            <Form.Item name="tlsKey" label="TLS Private Key (.key)" rules={[{ required: true, message: '비밀키 내용을 입력해주세요.' }]}>
              <TextArea rows={6} placeholder="-----BEGIN PRIVATE KEY-----..." style={{ fontFamily: 'monospace', fontSize: '13px' }} />
            </Form.Item>

            <div style={{ textAlign: 'right' }}>
              <Button type="primary" htmlType="submit" loading={loadingCert}>인증서 갱신하기</Button>
            </div>
          </Form>
        </Card>
      </Content>
    </Layout>
  );
};
