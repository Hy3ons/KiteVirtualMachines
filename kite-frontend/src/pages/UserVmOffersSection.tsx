import React, { useState } from 'react';
import { Alert, Button, Card, Empty, Form, Input, Modal, Space, Table, Tag, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import { CloudServerOutlined, GiftOutlined } from '@ant-design/icons';
import type { VmOffer } from '../api/types';
import type { VmOfferClaimFormValues } from './UserDashboardTypes';

const { Text } = Typography;

type UserVmOffersSectionProps = {
  readonly offers: readonly VmOffer[];
  readonly loading: boolean;
  readonly onClaim: (offer: VmOffer, values: VmOfferClaimFormValues) => Promise<void>;
};

export const UserVmOffersSection: React.FC<UserVmOffersSectionProps> = ({ offers, loading, onClaim }) => {
  const [form] = Form.useForm<VmOfferClaimFormValues>();
  const [selectedOffer, setSelectedOffer] = useState<VmOffer | null>(null);
  const [claiming, setClaiming] = useState(false);

  const handleClaim = async (values: VmOfferClaimFormValues) => {
    if (!selectedOffer) {
      return;
    }

    setClaiming(true);
    try {
      await onClaim(selectedOffer, values);
      form.resetFields();
      setSelectedOffer(null);
    } finally {
      setClaiming(false);
    }
  };

  const columns: TableColumnsType<VmOffer> = [
    { title: 'Offer', dataIndex: 'name', key: 'name', render: (name: string) => <Text strong>{name}</Text> },
    { title: 'Spec', key: 'spec', render: (_, offer) => `${offer.cpu} CPU / ${offer.memory} / ${offer.disk}` },
    { title: 'Image', dataIndex: 'image', key: 'image', render: (image: string) => <Tag>{image}</Tag> },
    { title: 'Expires', dataIndex: 'expiresAt', key: 'expiresAt', render: (value: string) => new Date(value).toLocaleString() },
    {
      title: 'Action',
      key: 'action',
      fixed: 'right',
      render: (_, offer) => (
        <Button type="primary" icon={<CloudServerOutlined />} onClick={() => setSelectedOffer(offer)}>
          Claim
        </Button>
      ),
    },
  ];

  return (
    <Card className="dashboard-offer-card">
      <div className="dashboard-offer-head">
        <Space>
          <GiftOutlined className="dashboard-offer-icon" />
          <div>
            <Text strong>Admin VM Offers</Text>
            <div className="dashboard-offer-copy">관리자가 미리 배정한 VM 스펙입니다. 이름과 로그인 정보를 채우면 내 VM으로 생성됩니다.</div>
          </div>
        </Space>
      </div>
      <Table
        columns={columns}
        dataSource={offers}
        rowKey="id"
        loading={loading}
        pagination={false}
        scroll={{ x: 860 }}
        locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="관리자가 배정한 VM offer가 없습니다." /> }}
      />

      <Modal
        title="Claim VM Offer"
        open={selectedOffer !== null}
        onCancel={() => setSelectedOffer(null)}
        footer={null}
        centered
      >
        {selectedOffer && (
          <Form form={form} layout="vertical" onFinish={handleClaim} className="vm-create-form">
            <Alert
              type="info"
              showIcon
              className="form-alert"
              message={`${selectedOffer.cpu} CPU / ${selectedOffer.memory} / ${selectedOffer.disk}`}
              description="관리자가 지정한 스펙은 claim 과정에서 바꿀 수 없습니다."
            />
            <Form.Item name="vmName" label="VM Name" rules={[{ required: true, message: 'VM 이름을 입력하세요.' }]}>
              <Input placeholder="project-runner" />
            </Form.Item>
            <Form.Item name="domainPrefix" label="Domain Prefix" rules={[{ required: true, whitespace: true, message: '도메인 prefix를 입력하세요.' }]}>
              <Input placeholder="project-runner" />
            </Form.Item>
            <Form.Item name="sshId" label="SSH User ID" rules={[{ required: true, message: '접속용 ID를 입력하세요.' }]}>
              <Input placeholder="ubuntu" />
            </Form.Item>
            <Form.Item
              name="initialLoginPassword"
              label="Initial Login Password"
              extra="VM 생성 시 정한 초기 비밀번호입니다. Kite에서 변경하거나 복구할 수 없습니다."
              rules={[{ required: true, message: '초기 로그인 비밀번호를 입력하세요.' }]}
            >
              <Input.Password placeholder="Strong password" />
            </Form.Item>
            <Form.Item className="form-actions">
              <Button onClick={() => setSelectedOffer(null)}>Cancel</Button>
              <Button type="primary" htmlType="submit" loading={claiming}>Claim Offer</Button>
            </Form.Item>
          </Form>
        )}
      </Modal>
    </Card>
  );
};
