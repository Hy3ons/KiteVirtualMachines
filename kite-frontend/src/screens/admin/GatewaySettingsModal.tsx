import { Button, Form, Input, Modal } from 'antd';
import { QuestionCircleOutlined } from '@ant-design/icons';
import type { TooltipProps } from 'antd';
import type { AdminSettingsState } from './useAdminSettings';

const portTooltip = {
  trigger: ['hover', 'focus', 'click'],
  placement: 'topLeft',
  styles: { root: { maxWidth: 'min(360px, calc(100vw - 32px))' } },
} satisfies TooltipProps;

type GatewaySettingsModalProps = {
  readonly settings: AdminSettingsState;
  readonly open: boolean;
  readonly onClose: () => void;
};

// GatewaySettingsModal edits the gateway ports using the existing Go-backed save handler.
// settings provides verified configuration and form state; onClose runs only after success or cancellation.
export function GatewaySettingsModal({ settings, open, onClose }: GatewaySettingsModalProps) {
  const { sshGateway, sshGatewayForm, handleSaveSSHGateway, loadingSSHGateway } = settings;
  const enabled = Boolean(sshGateway?.externalEnabled);

  // disableGateway preserves the saved ports while requesting external access removal.
  const disableGateway = async () => {
    if (await handleSaveSSHGateway({ externalEnabled: false, externalPort: sshGateway?.externalPort || '', publicPort: sshGateway?.publicPort || '' })) onClose();
  };

  return (
    <Modal title={enabled ? 'SSH Gateway 설정' : 'SSH Gateway 활성화'} open={open} onCancel={() => { if (!loadingSSHGateway) onClose(); }} closable={!loadingSSHGateway} keyboard={!loadingSSHGateway} footer={null} forceRender centered className="gateway-settings-modal">
      <Form form={sshGatewayForm} layout="vertical" disabled={loadingSSHGateway} onFinish={async (values) => {
        if (await handleSaveSSHGateway({ ...values, externalEnabled: true })) onClose();
      }}>
        <Form.Item name="externalPort" label="Gateway Service 포트" tooltip={{
          ...portTooltip,
          icon: <QuestionCircleOutlined tabIndex={0} aria-label="Gateway Service 포트 설명" />,
          title: <div className="gateway-port-help">
            <p>Kite Gateway는 외부 SSH 연결을 받아 각 VM으로 중계하는 SSH 프록시 서버입니다. 이 서버로 접속할 수 있도록 Kubernetes Service에 열 포트를 입력하세요.</p>
            <p>예: 12311. 같은 외부 접속 주소에서 다른 서비스가 이미 사용 중인 포트와 충돌하지 않는지 확인하세요.</p>
          </div>,
        }}>
          <Input placeholder="예: 12311 또는 40022" size="large" inputMode="numeric" />
        </Form.Item>

        <Form.Item name="publicPort" label="사용자 안내 포트" tooltip={{
          ...portTooltip,
          icon: <QuestionCircleOutlined tabIndex={0} aria-label="사용자 안내 포트 설명" />,
          title: <div className="gateway-port-help">
            <p>사용자가 SSH Gateway에 접속할 때 실제로 사용할 외부 포트입니다. 공유기나 라우터의 포트 포워딩 설정에 따라 Gateway Service 포트와 다를 수 있습니다.</p>
            <p>사용자 안내 포트는 22번을 권장합니다. 기존 서버의 SSH를 유지한다면 공유기에서 다음과 같이 설정할 수 있습니다.</p>
            <p><code>외부 2222 → 기존 서버의 22</code><code>외부 22 → Gateway Service 포트 12311</code></p>
            <p>이 예시에서는 Gateway Service 포트에 12311, 사용자 안내 포트에 22를 입력합니다. 실제 주소와 포트에 맞춰 공유기 설정도 직접 적용해야 하며, Kite는 기존 서버의 SSH 설정을 변경하지 않습니다.</p>
          </div>,
        }}>
          <Input placeholder="예: 공유기가 22 -> 12311로 전달하면 22" size="large" inputMode="numeric" />
        </Form.Item>
        <div className="gateway-modal-actions">
          {enabled && <Button danger disabled={loadingSSHGateway} onClick={() => void disableGateway()}>비활성화하기</Button>}
          <div><Button disabled={loadingSSHGateway} onClick={onClose}>취소</Button><Button type="primary" htmlType="submit" loading={loadingSSHGateway}>{enabled ? '변경사항 저장' : '활성화하기'}</Button></div>
        </div>
      </Form>
    </Modal>
  );
}
