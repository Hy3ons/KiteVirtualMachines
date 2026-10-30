import { useState } from 'react';
import { Alert, Button, Card, Descriptions, Typography } from 'antd';
import { CloudServerOutlined } from '@ant-design/icons';
import { StatusBadge } from '../../components/StatusBadge';
import { GatewaySettingsModal } from './GatewaySettingsModal';
import type { AdminSettingsState } from './useAdminSettings';

const { Title } = Typography;

// GatewaySettings shows observed gateway state and opens the port editor beside its status.
// settings owns the Go-backed configuration, shared form and loading state.
export function GatewaySettings({ settings }: { readonly settings: AdminSettingsState }) {
  const { sshGateway, sshGatewayForm, loadingSSHGateway } = settings;
  const [open, setOpen] = useState(false);
  const enabled = Boolean(sshGateway?.externalEnabled);

  // openEditor restores saved ports so cancelling and reopening discards unsaved edits.
  const openEditor = () => {
    sshGatewayForm.setFieldsValue({
      externalPort: sshGateway?.externalPort || '',
      publicPort: sshGateway?.publicPort || '22',
    });
    setOpen(true);
  };

  return (
    <section id="gateway" className="admin-settings-section" aria-labelledby="gateway-title">
      <Card className="admin-settings-card">
        <Title level={2} id="gateway-title"><CloudServerOutlined style={{ marginRight: 8 }} /> SSH Gateway</Title>
        <Alert
          type={sshGateway?.status.phase === 'Failed' ? 'error' : sshGateway?.status.phase === 'Blocked' ? 'warning' : 'info'}
          showIcon
          title={<div className="gateway-status-heading"><strong>현재 상태</strong><StatusBadge phase={sshGateway?.status.phase || 'Disabled'} /><Button size="small" type={enabled ? 'default' : 'primary'} onClick={openEditor} disabled={loadingSSHGateway}>{enabled ? '설정 변경' : '활성화하기'}</Button></div>}
          description={sshGateway?.status.message || '외부 VM SSH gateway가 아직 활성화되지 않았습니다.'}
        />
        {enabled && <Descriptions bordered size="small" column={{ xs: 1, sm: 1, md: 2 }} className="gateway-observed-settings">
          <Descriptions.Item label="Service 포트">{sshGateway?.externalPort || '-'}</Descriptions.Item>
          <Descriptions.Item label="사용자 안내 포트">{sshGateway?.publicPort || sshGateway?.externalPort || '-'}</Descriptions.Item>
          <Descriptions.Item label="적용된 Service 포트">{sshGateway?.status.observedExternalPort || '-'}</Descriptions.Item>
          <Descriptions.Item label="적용된 서비스">{sshGateway?.status.observedServiceName || '-'}</Descriptions.Item>
          <Descriptions.Item label="상태 사유">{sshGateway?.status.reason || '-'}</Descriptions.Item>
          <Descriptions.Item label="마지막 상태 변경">{sshGateway?.status.lastTransitionTime || '-'}</Descriptions.Item>
        </Descriptions>}
      </Card>
      <GatewaySettingsModal settings={settings} open={open} onClose={() => setOpen(false)} />
    </section>
  );
}
