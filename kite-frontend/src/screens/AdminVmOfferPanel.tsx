import React from 'react';
import { Alert, Button, Form, Input, InputNumber, Select, Space, Typography } from 'antd';
import { GiftOutlined } from '@ant-design/icons';
import type { AdminCreateVmOfferPayload, AdminUser } from '../api/types';

const { Title } = Typography;

type AdminVmOfferPanelProps = {
  readonly users: readonly AdminUser[];
  readonly loading: boolean;
  readonly onCreate: (payload: AdminCreateVmOfferPayload) => Promise<boolean>;
};

type OfferFormValues = {
  readonly targetUser: string;
  readonly cpu: number;
  readonly memory: string;
  readonly disk: string;
  readonly image?: string;
};

// AdminVmOfferPanel builds an allocation for users; loading and onCreate connect the administrator request state.
export const AdminVmOfferPanel: React.FC<AdminVmOfferPanelProps> = ({ users, loading, onCreate }) => {
  const [form] = Form.useForm<OfferFormValues>();
  const userOptions = users
    .filter((user) => user.namespace !== 'system')
    .map((user) => ({ label: `${user.username} (${user.namespace})`, value: user.username }));

  // handleFinish forwards validated allocation values and resets the completed form.
  const handleFinish = async (values: OfferFormValues) => {
    if (await onCreate(values)) form.resetFields();
  };

  return (
    <div className="admin-offer-card">
      <Space align="center" className="admin-offer-heading">
        <GiftOutlined className="admin-offer-icon" />
        <div>
          <Title level={3}>VM 사전 할당</Title>
        </div>
      </Space>
      <Alert
        type="info"
        showIcon
        className="form-alert"
        title="사용자가 수락하면 가상머신이 생성됩니다."
        description="사용자가 수락하기 전에는 VM이 생성되지 않습니다. 사용하지 않은 할당은 만료 후 정리되며, 기본 유효 시간은 24시간입니다."
      />
      <Form form={form} layout="vertical" onFinish={handleFinish} initialValues={{ cpu: 4, memory: '8Gi', disk: '30Gi', image: 'ubuntu-22.04' }}>
        <Form.Item name="targetUser" label="대상 사용자" rules={[{ required: true, message: '사용자를 선택하세요.' }]}>
          <Select placeholder="사용자 선택" options={userOptions} showSearch optionFilterProp="label" />
        </Form.Item>
        <div className="admin-offer-spec-row">
          <Form.Item name="cpu" label="CPU" rules={[{ required: true, message: 'CPU를 입력하세요.' }]}>
            <InputNumber min={1} max={64} />
          </Form.Item>
          <Form.Item name="memory" label="메모리" rules={[{ required: true, message: 'Memory를 입력하세요.' }]}>
            <Input placeholder="8Gi" />
          </Form.Item>
          <Form.Item name="disk" label="디스크" rules={[{ required: true, message: 'Disk를 입력하세요.' }]}>
            <Input placeholder="30Gi" />
          </Form.Item>
          <Form.Item name="image" label="이미지">
            <Input placeholder="ubuntu-22.04" />
          </Form.Item>
        </div>
        <div className="form-actions">
          <Button type="primary" htmlType="submit" loading={loading}>사전 할당 생성</Button>
        </div>
      </Form>
    </div>
  );
};
