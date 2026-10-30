import { Button, Card, Form, Input, Typography } from 'antd';
import type { ConfigResponse } from '../../api/types';
import type { AdminSettingsState } from './useAdminSettings';

const { Title, Paragraph } = Typography;

// ContactSettings renders its labelled settings group using shared form state and SSR defaults.
export function ContactSettings({ settings, initialData }: { readonly settings: AdminSettingsState; readonly initialData?: ConfigResponse | null }) {
  const { contactForm, handleSaveAdminContact, loadingContact } = settings;
  return (
    <>
      <section id="contact" className="admin-settings-section" aria-labelledby="contact-title">
        <Card className="admin-settings-card">
          <Title level={2} id="contact-title">관리자 연락처</Title>
          <Paragraph className="admin-section-description">
            VM 생성 권한이 없는 사용자에게 보여줄 연락 창구입니다. 이메일, 전화번호, Slack 채널, URL처럼 운영팀이 실제로 받는 값을 입력하세요.
          </Paragraph>
          <Form form={contactForm} initialValues={{ adminContact: initialData?.config.adminContact }} layout="vertical" onFinish={handleSaveAdminContact}>
            <Form.Item name="adminContact" label="관리자 연락처">
              <Input placeholder="ops@example.com / Slack #kite-help / https://..." size="large" />
            </Form.Item>
            <div style={{ textAlign: 'right' }}>
              <Button type="primary" htmlType="submit" loading={loadingContact}>연락처 저장</Button>
            </div>
          </Form>
        </Card>
      </section>

    </>
  );
}
