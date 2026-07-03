import React from 'react';
import { Alert, Button, Card, Form, Input, InputNumber, Select, Space, Typography } from 'antd';
import { GiftOutlined } from '@ant-design/icons';
import type { AdminCreateVmOfferPayload, AdminUser } from '../api/types';

const { Paragraph, Title } = Typography;

type AdminVmOfferPanelProps = {
  readonly users: readonly AdminUser[];
  readonly loading: boolean;
  readonly onCreate: (payload: AdminCreateVmOfferPayload) => Promise<void>;
};

type OfferFormValues = {
  readonly targetUser: string;
  readonly cpu: number;
  readonly memory: string;
  readonly disk: string;
  readonly image?: string;
};

export const AdminVmOfferPanel: React.FC<AdminVmOfferPanelProps> = ({ users, loading, onCreate }) => {
  const [form] = Form.useForm<OfferFormValues>();
  const userOptions = users
    .filter((user) => user.namespace !== 'system')
    .map((user) => ({ label: `${user.username} (${user.namespace})`, value: user.username }));

  const handleFinish = async (values: OfferFormValues) => {
    await onCreate(values);
    form.resetFields();
  };

  return (
    <Card className="admin-offer-card">
      <Space align="start" className="admin-offer-heading">
        <GiftOutlined className="docs-card-icon" />
        <div>
          <Title level={3}>VM Offer 생성</Title>
          <Paragraph>
            Level 3 관리자가 사용자에게 스펙 일부를 미리 배정합니다. 사용자는 대시보드에서 이름과 초기 로그인 정보를 채워 claim합니다.
          </Paragraph>
        </div>
      </Space>
      <Alert
        type="info"
        showIcon
        className="form-alert"
        message="Offer는 실제 VM이 아닙니다."
        description="Claim 전까지는 KiteVirtualMachine이 만들어지지 않으며, 사용하지 않은 offer는 controller가 만료 시간 이후 정리합니다. 기본 TTL은 24시간입니다."
      />
      <Form form={form} layout="vertical" onFinish={handleFinish} initialValues={{ cpu: 4, memory: '8Gi', disk: '30Gi', image: 'ubuntu-22.04' }}>
        <Form.Item name="targetUser" label="Target User" rules={[{ required: true, message: '사용자를 선택하세요.' }]}>
          <Select placeholder="사용자 선택" options={userOptions} showSearch optionFilterProp="label" />
        </Form.Item>
        <Space size={12} wrap className="admin-offer-spec-row">
          <Form.Item name="cpu" label="CPU" rules={[{ required: true, message: 'CPU를 입력하세요.' }]}>
            <InputNumber min={1} max={64} />
          </Form.Item>
          <Form.Item name="memory" label="Memory" rules={[{ required: true, message: 'Memory를 입력하세요.' }]}>
            <Input placeholder="8Gi" />
          </Form.Item>
          <Form.Item name="disk" label="Disk" rules={[{ required: true, message: 'Disk를 입력하세요.' }]}>
            <Input placeholder="30Gi" />
          </Form.Item>
          <Form.Item name="image" label="Image">
            <Input placeholder="ubuntu-22.04" />
          </Form.Item>
        </Space>
        <div className="form-actions">
          <Button type="primary" htmlType="submit" loading={loading}>Create Offer</Button>
        </div>
      </Form>
    </Card>
  );
};
