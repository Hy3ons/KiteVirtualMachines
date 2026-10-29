import { Button, Card, Form, Input, Space, Typography } from 'antd';
import { ReloadOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import type { AdminSettingsState } from './useAdminSettings';

const { Title, Paragraph, Text } = Typography;
const { TextArea } = Input;

// SecuritySettings renders its labelled settings group using shared form state and secret rotation actions.
export function SecuritySettings({ settings }: { readonly settings: AdminSettingsState }) {
  const { runtimeStatus, handleRotateRuntimeSecrets, loadingRuntime, certForm, handleSaveCert, loadingCert } = settings;
  return (
    <>
      <section id="runtime" className="admin-settings-section" aria-labelledby="runtime-title">
        <Card className="admin-settings-card">
          <Title level={2} id="runtime-title">API 런타임 설정</Title>
          <Paragraph className="admin-section-description">
            JWT Secret과 Password Salt는 서버 최초 시작 시 자동 생성되어 Kubernetes Secret에 저장됩니다. 원문은 화면에 표시하지 않습니다.
            <br />재생성된 값은 저장 후 kite-api 재시작 시 적용됩니다.
          </Paragraph>

          <Space wrap style={{ marginBottom: 16 }}>
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
      </section>

      <section id="certificates" className="admin-settings-section" aria-labelledby="certificates-title">
        <Card className="admin-settings-card">
          <Title level={2} id="certificates-title"><SafetyCertificateOutlined style={{ marginRight: 8 }} /> 와일드카드 인증서 갱신/등록</Title>
          <Paragraph className="admin-section-description">
            등록된 도메인의 서브도메인을 모두 커버하는 TLS 인증서를 갱신합니다. 저장 시 즉시 K8s의 <Text strong>kite/global-tls-secret</Text>이 교체됩니다.
          </Paragraph>
          
          <Form form={certForm} layout="vertical" onFinish={handleSaveCert}>
            <Form.Item name="tlsCert" label="TLS Fullchain Certificate (.crt / .pem)" rules={[{ required: true, message: '인증서 내용을 입력해주세요.' }]}>
              <TextArea rows={6} placeholder="-----BEGIN CERTIFICATE-----..." className="admin-pem-input" />
            </Form.Item>

            <Form.Item name="tlsKey" label="TLS Private Key (.key)" rules={[{ required: true, message: '비밀키 내용을 입력해주세요.' }]}>
              <TextArea rows={6} placeholder="-----BEGIN PRIVATE KEY-----..." className="admin-pem-input" />
            </Form.Item>

            <div style={{ textAlign: 'right' }}>
              <Button type="primary" htmlType="submit" loading={loadingCert}>인증서 갱신하기</Button>
            </div>
          </Form>
        </Card>
      </section>
    </>
  );
}
