import React from 'react';
import { Alert, Button, Form, Input, InputNumber, Modal, Typography } from 'antd';
import { MOCK_ENV } from '../config/mockEnv';
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
};

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
}) => (
  <Modal
    title={<Title level={4} className="modal-title">Create Virtual Machine</Title>}
    open={open}
    onCancel={onCancel}
    footer={null}
    centered
  >
    <Form form={form} layout="vertical" onFinish={onCreate} className="vm-create-form">
      {accessLevel === 0 && (
        <Alert
          title="VM 생성 권한 없음"
          description="Level 0 계정은 VM을 생성할 수 없습니다. 관리자에게 권한을 요청하세요."
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

      <Form.Item name="domainPrefix" label="Domain Prefix" extra={`.${MOCK_ENV.BASE_DOMAIN} (예시)`}>
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

      <Form.Item name="disk" label="Disk Size (Gi)" initialValue={accessLevel === 1 ? fixedDiskGi : minDiskGi} rules={[{ required: true }]}>
        <InputNumber min={minDiskGi} max={100} disabled={accessLevel === 1} className="full-width" />
      </Form.Item>

      <div className="form-note">
        {accessLevel === 1
          ? '* Level 1 계정은 CPU 2, Memory 4Gi, Disk 20Gi, OS Image ubuntu-22.04로 고정 생성됩니다.'
          : '* 최소 Disk는 20Gi입니다. CPU, Memory, OS Image를 입력하지 않으면 기본값 CPU 2, Memory 4Gi, ubuntu-22.04가 적용됩니다.'}
      </div>

      <Form.Item className="form-actions">
        <Button onClick={onCancel}>Cancel</Button>
        <Button type="primary" htmlType="submit" disabled={!canCreateVm}>Create</Button>
      </Form.Item>
    </Form>
  </Modal>
);
