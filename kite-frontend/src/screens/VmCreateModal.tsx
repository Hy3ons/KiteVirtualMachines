import React from 'react';
import { Alert, Button, Form, Input, InputNumber, Modal, Typography } from 'antd';
import { UI_EXAMPLE_BASE_DOMAIN } from '../config/uiDefaults';
import type { VmCreateForm, VmCreateFormValues } from './UserDashboardTypes';

const { Title } = Typography;

type VmCreateModalProps = {
  readonly open: boolean;
  readonly onCancel: () => void;
  readonly onCreate: (values: VmCreateFormValues) => Promise<void>;
  readonly form: VmCreateForm;
  readonly canCreateVm: boolean;
  readonly accessLevel: number;
  readonly fixedCpu: number;
  readonly fixedMemory: string;
  readonly fixedDiskGi: number;
  readonly minDiskGi: number;
  readonly adminContact: string;
  readonly baseDomain: string;
};

// VmCreateModal collects VM identity and credentials while displaying the access-level specification limits.
export const VmCreateModal: React.FC<VmCreateModalProps> = ({
  open,
  onCancel,
  onCreate,
  form,
  canCreateVm,
  accessLevel,
  fixedCpu,
  fixedMemory,
  fixedDiskGi,
  minDiskGi,
  adminContact,
  baseDomain,
}) => {
  const domainExample = baseDomain.trim() || UI_EXAMPLE_BASE_DOMAIN;

  return (
    <Modal
      title={<Title level={4} className="modal-title">가상머신 만들기</Title>}
      className="workspace-vm-modal"
      open={open}
      onCancel={onCancel}
      footer={null}
      centered
    >
      <Form form={form} layout="vertical" onFinish={onCreate} className="vm-create-form">
        {accessLevel === 0 && (
          <Alert
            title="VM 생성 권한 없음"
            description={`Level 0 계정은 VM을 직접 생성할 수 없습니다. 관리자에게 권한을 요청하세요.${adminContact ? ` 연락처: ${adminContact}` : ''}`}
            type="warning"
            showIcon
            className="form-alert"
          />
        )}
        {accessLevel === 1 && (
          <Alert
            title="일반 계정 VM 스펙 고정"
            description={`Level 1 계정은 CPU ${fixedCpu}, RAM ${fixedMemory}, Disk ${fixedDiskGi}Gi로 생성됩니다.`}
            type="info"
            showIcon
            className="form-alert"
          />
        )}

        <Form.Item name="name" label="VM Name" rules={[{ required: true, message: '영문 소문자/숫자 조합의 이름을 입력하세요.' }]}>
          <Input placeholder="my-ubuntu-vm" />
        </Form.Item>

        <Form.Item
          name="domainPrefix"
          label="Domain Prefix"
          extra={`.${domainExample} (예시)`}
          rules={[{ required: true, whitespace: true, message: '도메인 prefix를 입력하세요.' }]}
        >
          <Input placeholder="my-web" />
        </Form.Item>

        <Form.Item name="sshId" label="SSH User ID" rules={[{ required: true, message: '접속용 ID를 입력하세요.' }]}>
          <Input placeholder="ubuntu" />
        </Form.Item>

        <Form.Item
          name="sshPassword"
          label="Initial Login Password"
          extra="VM 생성 시 한 번 설정됩니다. SSH gateway 로그인과 웹 콘솔 OS 로그인에 함께 사용되며, Kite에서 변경하거나 복구할 수 없습니다."
          rules={[{ required: true, message: '초기 로그인 비밀번호를 입력하세요.' }]}
        >
          <Input.Password placeholder="Strong password" />
        </Form.Item>

        <Form.Item
          name="disk"
          label="디스크 용량 (GiB)"
          initialValue={accessLevel === 1 ? fixedDiskGi : minDiskGi}
          extra={accessLevel === 1 ? `일반 계정이 직접 생성하는 VM의 디스크는 ${fixedDiskGi}GiB로 고정됩니다. 더 큰 디스크가 필요하면 관리자에게 요청하세요.${adminContact ? ` 관리자 연락처: ${adminContact}` : ''}` : undefined}
          rules={[{ required: true }]}
        >
          <InputNumber min={accessLevel === 1 ? fixedDiskGi : minDiskGi} max={accessLevel === 1 ? fixedDiskGi : 100} disabled={accessLevel === 1} className="full-width" />
        </Form.Item>

        {accessLevel !== 1 && <div className="form-note">
          {`최소 디스크는 ${minDiskGi}GiB입니다. CPU, Memory, OS Image를 입력하지 않으면 기본값 CPU 2, Memory 4Gi, ubuntu-22.04가 적용됩니다.`}
        </div>}

        <Form.Item className="form-actions">
          <Button onClick={onCancel}>취소</Button>
          <Button type="primary" htmlType="submit" disabled={!canCreateVm}>가상머신 만들기</Button>
        </Form.Item>
      </Form>
    </Modal>
  );
};
