import React from 'react';
import { Avatar, Card, Col, Progress, Row, Space, Statistic, Typography } from 'antd';
import { CloudServerOutlined, UserOutlined } from '@ant-design/icons';
import type { DashboardVm } from './UserDashboardTypes';

const { Title, Text } = Typography;

type UserDashboardSummaryProps = {
  readonly username: string | null;
  readonly namespace: string | null;
  readonly profileImage: string | null;
  readonly accessLevel: number;
  readonly vms: readonly DashboardVm[];
  readonly quotaLimit: number;
  readonly accessDescription: string;
};

export const UserDashboardSummary: React.FC<UserDashboardSummaryProps> = ({
  username,
  namespace,
  profileImage,
  accessLevel,
  vms,
  quotaLimit,
  accessDescription,
}) => {
  const runningVmCount = vms.filter((vm) => vm.phase === 'Running').length;
  const stoppedVmCount = vms.filter((vm) => vm.phase === 'Stopped').length;
  const pendingVmCount = vms.filter((vm) => vm.phase === 'Creating' || vm.phase === 'Terminating').length;
  const quotaPercent = accessLevel === 1 ? Math.min(Math.round((vms.length / quotaLimit) * 100), 100) : 0;

  return (
    <>
      <Row gutter={[24, 24]} className="dashboard-summary-row">
        <Col xs={24} lg={14}>
          <Card hoverable className="dashboard-profile-card">
            <div className="dashboard-profile">
              <div className="dashboard-profile-main">
                <Avatar src={profileImage || '/default_profile.png'} icon={<UserOutlined />} size={48} />
                <div className="dashboard-profile-copy">
                  <Title level={4} className="dashboard-card-title">
                    {username || 'debug'}님의 프로필
                  </Title>
                  <Text type="secondary">{accessDescription}</Text>
                </div>
              </div>
              <Space size="large" wrap className="dashboard-profile-meta">
                <div>
                  <Text type="secondary" className="dashboard-meta-label">NAMESPACE</Text>
                  <div className="dashboard-meta-value">{namespace || 'system'}</div>
                </div>
                <div>
                  <Text type="secondary" className="dashboard-meta-label">ACCESS LEVEL</Text>
                  <div className="dashboard-meta-value">Level {accessLevel}</div>
                </div>
                <div>
                  <Text type="secondary" className="dashboard-meta-label">VM QUOTA</Text>
                  <div className="dashboard-meta-value dashboard-meta-accent">
                    {accessLevel === 0 ? 'No Access' : accessLevel === 1 ? `Max ${quotaLimit} VMs` : 'Unlimited'}
                  </div>
                </div>
              </Space>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={8} lg={4}>
          <Card hoverable>
            <Statistic title="VMs" value={vms.length} prefix={<CloudServerOutlined />} styles={{ content: { color: '#8B7355' } }} />
          </Card>
        </Col>
        <Col xs={24} sm={8} lg={3}>
          <Card hoverable>
            <Statistic title="Running" value={runningVmCount} styles={{ content: { color: '#7A9C74' } }} />
          </Card>
        </Col>
        <Col xs={24} sm={8} lg={3}>
          <Card hoverable>
            <Statistic
              title={pendingVmCount > 0 ? 'Pending' : 'Stopped'}
              value={pendingVmCount > 0 ? pendingVmCount : stoppedVmCount}
              styles={{ content: { color: pendingVmCount > 0 ? '#D4A373' : '#C86B6B' } }}
            />
          </Card>
        </Col>
      </Row>

      {accessLevel === 1 && (
        <Card className="dashboard-quota-card">
          <Space orientation="vertical" className="full-width" size={8}>
            <div className="dashboard-quota-head">
              <Text strong>Level 1 VM quota</Text>
              <Text type="secondary">{vms.length} / {quotaLimit}</Text>
            </div>
            <Progress percent={quotaPercent} strokeColor="#8B7355" showInfo={false} />
            <Text type="secondary">CPU 2, RAM 4Gi, Disk 20Gi 스펙으로만 생성됩니다.</Text>
          </Space>
        </Card>
      )}
    </>
  );
};
