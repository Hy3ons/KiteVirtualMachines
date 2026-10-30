import React, { useState } from 'react';
import { Alert, Button, Empty, Form, Input, Modal, Table, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import { GiftOutlined } from '@ant-design/icons';
import type { VmOffer } from '../api/types';
import type { VmOfferClaimFormValues } from './UserDashboardTypes';

const { Text } = Typography;
const expiryFormatter = new Intl.DateTimeFormat('ko-KR', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'Asia/Seoul' });

// formatExpiry shows an offer's expiresAt in Korean local time, with a safe fallback for missing dates.
function formatExpiry(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '—' : expiryFormatter.format(date);
}

type UserVmOffersSectionProps = {
  readonly offers: readonly VmOffer[];
  readonly loading: boolean;
  readonly onClaim: (offer: VmOffer, values: VmOfferClaimFormValues) => Promise<boolean>;
};

// UserVmOffersSection lists assigned offers and submits validated claim credentials through onClaim.
export const UserVmOffersSection: React.FC<UserVmOffersSectionProps> = ({ offers, loading, onClaim }) => {
  const [form] = Form.useForm<VmOfferClaimFormValues>();
  const [selectedOffer, setSelectedOffer] = useState<VmOffer | null>(null);
  const [claiming, setClaiming] = useState(false);

  // handleClaim submits values for the chosen offer and clears its form after the API action completes.
  const handleClaim = async (values: VmOfferClaimFormValues) => {
    if (!selectedOffer) {
      return;
    }

    setClaiming(true);
    try {
      if (await onClaim(selectedOffer, values)) {
        form.resetFields();
        setSelectedOffer(null);
      }
    } finally {
      setClaiming(false);
    }
  };

  const columns: TableColumnsType<VmOffer> = [
    { title: '배정 이름', key: 'name', width: 200, render: (_, offer) => <div className="workspace-offer-identity"><Text strong>{offer.name}</Text><span title={offer.image}>{offer.image}</span></div> },
    { title: '사양', key: 'spec', width: 184, render: (_, offer) => <div className="workspace-offer-spec"><span>{offer.cpu} vCPU</span><small>메모리 {offer.memory} · 디스크 {offer.disk}</small></div> },
    { title: '만료일', dataIndex: 'expiresAt', key: 'expiresAt', width: 168, render: formatExpiry },
    {
      title: '작업',
      key: 'action',
      width: 128,
      align: 'right',
      render: (_, offer) => (
        <Button type="primary" onClick={() => setSelectedOffer(offer)}>
          할당받기
        </Button>
      ),
    },
  ];

  return (
    <section id="offers" className="workspace-resource-panel workspace-offers" aria-labelledby="offers-title">
      <div className="workspace-section-heading workspace-offer-heading">
        <GiftOutlined className="workspace-offer-icon" />
        <h2 id="offers-title">배정된 가상머신</h2>
      </div>
      <Table
        columns={columns}
        dataSource={offers}
        rowKey="id"
        loading={loading}
        pagination={false}
        scroll={{ x: 680 }}
        locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="관리자가 배정한 가상머신이 없습니다." /> }}
      />

      <Modal
        title="배정된 가상머신 만들기"
        className="workspace-vm-modal workspace-vm-offer-modal"
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
              title={`${selectedOffer.cpu} CPU / ${selectedOffer.memory} / ${selectedOffer.disk}`}
              description="관리자가 지정한 스펙은 claim 과정에서 바꿀 수 없습니다."
            />
            <Form.Item name="vmName" label="가상머신 이름" rules={[{ required: true, message: 'VM 이름을 입력하세요.' }]}>
              <Input placeholder="project-runner" />
            </Form.Item>
            <Form.Item name="domainPrefix" label="도메인 접두사" rules={[{ required: true, whitespace: true, message: '도메인 접두사를 입력하세요.' }]}>
              <Input placeholder="project-runner" />
            </Form.Item>
            <Form.Item name="sshId" label="SSH 사용자 ID" rules={[{ required: true, message: '접속용 ID를 입력하세요.' }]}>
              <Input placeholder="ubuntu" />
            </Form.Item>
            <Form.Item
              name="initialLoginPassword"
              label="초기 비밀번호"
              extra="VM 생성 시 정한 초기 비밀번호입니다. Kite에서 변경하거나 복구할 수 없습니다."
              rules={[{ required: true, message: '초기 로그인 비밀번호를 입력하세요.' }]}
            >
              <Input.Password placeholder="Strong password" />
            </Form.Item>
            <Form.Item className="form-actions">
              <Button onClick={() => setSelectedOffer(null)}>취소</Button>
              <Button type="primary" htmlType="submit" loading={claiming}>할당받기</Button>
            </Form.Item>
          </Form>
        )}
      </Modal>
    </section>
  );
};
