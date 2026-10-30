import { Button, Card, Form, Input, Space, Switch, Typography } from 'antd';
import { GlobalOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { UI_EXAMPLE_BASE_DOMAIN } from '../../config/uiDefaults';
import type { ConfigResponse } from '../../api/types';
import type { AdminSettingsState } from './useAdminSettings';

const { Title, Paragraph, Text } = Typography;

// RoutingSettings renders its labelled settings group using shared form state and SSR defaults.
export function RoutingSettings({ settings, initialData }: { readonly settings: AdminSettingsState; readonly initialData?: ConfigResponse | null }) {
  const { domainForm, loadingDomain, handleSaveDomain, forceHTTPS, loadingHTTPS, handleToggleForceHTTPS } = settings;
  return (
    <>
      <section id="domain" className="admin-settings-section" aria-labelledby="domain-title">
        <Card className="admin-settings-card">
          <Title level={2} id="domain-title"><GlobalOutlined style={{ marginRight: 8 }} /> 베이스 도메인 설정</Title>
          <Paragraph className="admin-section-description">
            Kite 플랫폼 Ingress의 host와 가상머신별 Ingress 도메인에 사용할 기본 도메인을 설정합니다.
            <br />도메인 변경 시 컨트롤러가 platform Ingress를 갱신하고 VM Ingress는 다음 reconcile에서 새 도메인으로 맞춰집니다.
          </Paragraph>

          <Form form={domainForm} initialValues={{ baseDomain: initialData?.config.baseDomain }} layout="vertical" onFinish={handleSaveDomain}>
            <Form.Item name="baseDomain" label="베이스 도메인" rules={[{ required: true, message: '도메인을 입력하세요.' }]}>
              <Input placeholder={UI_EXAMPLE_BASE_DOMAIN} size="large" />
            </Form.Item>
            <div style={{ textAlign: 'right' }}>
              <Button type="primary" htmlType="submit" loading={loadingDomain}>도메인 변경 및 전체 적용</Button>
            </div>
          </Form>
        </Card>
      </section>

      <section id="https" className="admin-settings-section" aria-labelledby="https-title">
        <Card className="admin-settings-card">
          <Title level={2} id="https-title"><SafetyCertificateOutlined style={{ marginRight: 8 }} /> 플랫폼 HTTPS 강제</Title>
          <Paragraph className="admin-section-description">
            플랫폼 Ingress에 Traefik HTTPS redirect middleware를 연결합니다. 내부 80 포트 터널 QA가 필요할 때는 꺼두고,
            공개 도메인을 HTTPS로 운영할 때 켜세요.
          </Paragraph>
          <Space align="center" size={16} wrap>
            <Switch aria-label="플랫폼 HTTPS 강제" checked={forceHTTPS} loading={loadingHTTPS} onChange={handleToggleForceHTTPS} />
            <Text strong>{forceHTTPS ? 'HTTP 요청을 HTTPS로 강제 전환' : 'HTTP 접속 허용'}</Text>
          </Space>
        </Card>
      </section>

    </>
  );
}
