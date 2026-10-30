import { Alert, Button, Card, Form, Input, Popconfirm, Typography } from 'antd';
import { ReloadOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { StatusBadge } from '../../components/StatusBadge';
import type { AdminSettingsState } from './useAdminSettings';

const { Title, Paragraph, Text } = Typography;
const { TextArea } = Input;
const passwordSaltWarning = 'Password Salt를 바꾸면 기존 Salt를 사용하는 구형 계정의 로그인과 구형 VM SSH 비밀번호 인증이 실패할 수 있습니다. 현재 bcrypt 형식의 비밀번호는 영향을 받지 않습니다.';
type SecuritySettingsState = Pick<AdminSettingsState, 'runtimeStatus' | 'handleRotateRuntimeSecrets' | 'loadingRuntime' | 'certForm' | 'handleSaveCert' | 'loadingCert'>;

// RuntimeSecretStatus pairs a secret name with its presence badge in the runtime settings list.
// present is null until the API has supplied a status; secret values are never rendered.
function RuntimeSecretStatus({ name, present }: { readonly name: 'JWT Secret' | 'Password Salt'; readonly present: boolean | null }) {
  return (
    <div className="runtime-secret-row">
      <dt>{name}</dt>
      <dd><StatusBadge phase={present === null ? 'Pending' : present ? 'Ready' : 'Failed'} label={present === null ? '확인 중' : present ? '생성됨' : '없음'} /></dd>
    </div>
  );
}

// SecuritySettings renders its labelled settings group using shared form state and secret rotation actions.
export function SecuritySettings({ settings }: { readonly settings: SecuritySettingsState }) {
  const { runtimeStatus, handleRotateRuntimeSecrets, loadingRuntime, certForm, handleSaveCert, loadingCert } = settings;
  return (
    <>
      <section id="runtime" className="admin-settings-section" aria-labelledby="runtime-title">
        <Card className="admin-settings-card">
          <Title level={2} id="runtime-title">API 런타임 설정</Title>
          <Paragraph className="admin-section-description">
            JWT Secret과 Password Salt는 서버 최초 시작 시 자동 생성되어 Kubernetes Secret에 저장됩니다. 원문은 화면에 표시하지 않습니다.
            <br />재생성된 값은 저장 후 kite-api 재시작 시 적용됩니다.
            <br />JWT Secret을 재생성하면 적용 후 기존 로그인 토큰이 모두 무효화됩니다.
          </Paragraph>
          <Alert className="form-alert" type="warning" showIcon title="Password Salt 재생성 주의" description={passwordSaltWarning} />

          <dl className="runtime-secret-status" aria-label="런타임 시크릿 생성 상태" aria-live="polite" aria-busy={runtimeStatus === null}>
            <RuntimeSecretStatus name="JWT Secret" present={runtimeStatus?.hasJWTSecret ?? null} />
            <RuntimeSecretStatus name="Password Salt" present={runtimeStatus?.hasPasswordSalt ?? null} />
          </dl>

          <div className="runtime-secret-actions">
            <Button
              icon={<ReloadOutlined />}
              loading={loadingRuntime}
              onClick={() => handleRotateRuntimeSecrets({ rotateJWTSecret: true })}
            >
              JWT Secret 재생성
            </Button>
            <Popconfirm classNames={{ root: 'runtime-secret-confirm' }} title="Password Salt를 재생성할까요?" description={passwordSaltWarning} okText="재생성" cancelText="취소" okButtonProps={{ danger: true }} disabled={loadingRuntime} onConfirm={() => handleRotateRuntimeSecrets({ rotatePasswordSalt: true })}>
              <Button icon={<ReloadOutlined />} loading={loadingRuntime}>Password Salt 재생성</Button>
            </Popconfirm>
            <Popconfirm classNames={{ root: 'runtime-secret-confirm' }} title="JWT Secret과 Password Salt를 모두 재생성할까요?" description={`${passwordSaltWarning} 기존 로그인 토큰도 모두 무효화됩니다.`} okText="모두 재생성" cancelText="취소" okButtonProps={{ danger: true }} disabled={loadingRuntime} onConfirm={() => handleRotateRuntimeSecrets({ rotateJWTSecret: true, rotatePasswordSalt: true })}>
              <Button danger icon={<ReloadOutlined />} loading={loadingRuntime}>둘 다 재생성</Button>
            </Popconfirm>
          </div>

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
