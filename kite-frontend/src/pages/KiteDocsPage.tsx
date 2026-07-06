import React from 'react';
import { SEO } from '../components/SEO';
import { GlobalHeader } from '../components/GlobalHeader';
import { Alert, Button, Card, Col, Divider, Layout, Row, Space, Tag, Timeline, Typography } from 'antd';
import {
  ApartmentOutlined,
  BookOutlined,
  CloudServerOutlined,
  ControlOutlined,
  GiftOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
  UserSwitchOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { accessLevelDocs, architectureTimelineItems } from './KiteDocsData';

const { Content } = Layout;
const { Title, Paragraph, Text } = Typography;

export const KiteDocsPage: React.FC = () => {
  const navigate = useNavigate();

  return (
    <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
      <SEO
        title="Kite Docs - 권한 모델과 아키텍처"
        description="Kite 권한 레벨, VM 생성 제한, Frontend, API, CRD, controller, KubeVirt 흐름과 SSH Gateway 운영 주의를 정리한 문서입니다."
        url="/kite-docs"
      />
      <GlobalHeader />

      <Content className="app-main app-main--wide docs-content">
        <section className="docs-hero">
          <Space size={8} wrap className="docs-kicker">
            <Tag>Access Control</Tag>
            <Tag>Architecture</Tag>
            <Tag>SSH Gateway</Tag>
            <Tag>KubeVirt</Tag>
          </Space>
          <Title level={1} className="docs-title">Kite Docs</Title>
          <Paragraph className="docs-subtitle">
            Kite의 권한 모델, VM 생성 제한, Frontend와 API, controller가 함께 VM을 만드는 흐름을 한 곳에 정리합니다.
            현재 적용된 정책과 운영 중 주의해야 할 한계를 함께 안내합니다.
          </Paragraph>
          <Space size={12} wrap>
            <Button type="primary" icon={<CloudServerOutlined />} onClick={() => navigate('/dashboard')}>
              Dashboard
            </Button>
            <Button icon={<BookOutlined />} href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank">
              Repository
            </Button>
          </Space>
        </section>

        <section className="docs-section">
          <div className="docs-section-heading">
            <Title level={2}>권한 레벨</Title>
            <Text type="secondary">Kite는 숫자가 높을수록 더 넓은 작업을 허용하되, Level 2와 Level 3의 권한 범위를 분리합니다.</Text>
          </div>
          <Row gutter={[16, 16]}>
            {accessLevelDocs.map((item) => (
              <Col xs={24} md={12} xl={6} key={item.level}>
                <Card className="docs-access-card" hoverable>
                  <div className="docs-access-level">Level {item.level}</div>
                  <Title level={4}>{item.title}</Title>
                  <Tag className="docs-access-tag">{item.tone}</Tag>
                  <Paragraph className="docs-card-summary">{item.summary}</Paragraph>
                  <ul className="docs-list">
                    {item.rules.map((rule) => (
                      <li key={rule}>{rule}</li>
                    ))}
                  </ul>
                </Card>
              </Col>
            ))}
          </Row>
        </section>

        <section className="docs-section">
          <Row gutter={[20, 20]} align="stretch">
            <Col xs={24} lg={8}>
              <Card className="docs-policy-card" hoverable>
                <SafetyCertificateOutlined className="docs-card-icon" />
                <Title level={3}>Level 1 제한</Title>
                <Paragraph>
                  일반 사용자는 직접 VM을 만들 수 있지만 스펙과 생성 한도가 고정됩니다. Frontend와 API가 모두 2대 초과 생성을 막습니다.
                </Paragraph>
                <div className="docs-policy-grid">
                  <span>CPU</span><strong>2</strong>
                  <span>Memory</span><strong>4Gi</strong>
                  <span>Disk</span><strong>25Gi</strong>
                  <span>Quota</span><strong>2 VMs</strong>
                </div>
              </Card>
            </Col>
            <Col xs={24} lg={8}>
              <Card className="docs-policy-card" hoverable>
                <UserSwitchOutlined className="docs-card-icon" />
                <Title level={3}>Level 2 범위</Title>
                <Paragraph>
                  Level 2는 자기 VM을 자유 스펙으로 만들 수 있지만, admin API에서는 사용자 권한 0과 1만 조정합니다.
                </Paragraph>
                <Alert type="info" showIcon title="전체 VM 제어, settings, 사용자 삭제, offer 생성은 Level 3 관리자 작업입니다." />
              </Card>
            </Col>
            <Col xs={24} lg={8}>
              <Card className="docs-policy-card" hoverable>
                <LockOutlined className="docs-card-icon" />
                <Title level={3}>Level 0 제한</Title>
                <Paragraph>
                  Level 0 사용자는 로그인할 수 있지만 VM 직접 생성과 제어 작업은 열리지 않습니다. Dashboard는 관리자 연락처를 보여줍니다.
                </Paragraph>
                <Alert type="warning" showIcon title="관리자가 배정한 VM Offer는 직접 생성 제한과 분리된 명시적 배정 흐름입니다." />
              </Card>
            </Col>
          </Row>
        </section>

        <section className="docs-section">
          <div className="docs-section-heading">
            <Title level={2}>VM Offer 흐름</Title>
            <Text type="secondary">관리자가 스펙을 미리 배정하되, 실제 VM 생성은 사용자가 claim할 때만 일어납니다.</Text>
          </div>
          <Row gutter={[20, 20]}>
            <Col xs={24} lg={8}>
              <Card className="docs-policy-card" hoverable>
                <GiftOutlined className="docs-card-icon" />
                <Title level={3}>분리된 CRD</Title>
                <Paragraph>
                  Offer는 `KiteVirtualMachineOffer`로 저장됩니다. CPU, memory, disk, image, expiresAt만 보관하고 실제 VM 리소스는 만들지 않습니다.
                </Paragraph>
              </Card>
            </Col>
            <Col xs={24} lg={8}>
              <Card className="docs-policy-card" hoverable>
                <UserSwitchOutlined className="docs-card-icon" />
                <Title level={3}>사용자 Claim</Title>
                <Paragraph>
                  사용자는 자기 namespace의 offer만 볼 수 있습니다. VM name, domain prefix, SSH ID, 초기 비밀번호를 채우면 offer 스펙과 합쳐 VM이 생성됩니다.
                </Paragraph>
              </Card>
            </Col>
            <Col xs={24} lg={8}>
              <Card className="docs-policy-card" hoverable>
                <SafetyCertificateOutlined className="docs-card-icon" />
                <Title level={3}>만료 정리</Title>
                <Paragraph>
                  기본 TTL은 24시간입니다. claim되지 않은 offer는 controller가 주기적으로 정리해 더미 배정값이 남지 않게 합니다.
                </Paragraph>
              </Card>
            </Col>
          </Row>
        </section>

        <section className="docs-section docs-architecture-section">
          <div className="docs-section-heading">
            <Title level={2}>아키텍처</Title>
            <Text type="secondary">Kite는 화면이 직접 VM을 만들지 않고, API와 CRD, controller를 거쳐 실제 KubeVirt 리소스를 맞춥니다.</Text>
          </div>
          <Row gutter={[24, 24]}>
            <Col xs={24} lg={10}>
              <Card className="docs-architecture-card" hoverable>
                <ApartmentOutlined className="docs-card-icon" />
                <Title level={3}>역할을 나누는 이유</Title>
                <Paragraph>
                  Kite는 Frontend가 KubeVirt를 직접 제어하지 않도록 역할을 나눕니다.
                  API는 권한과 입력을 확인하고, controller는 CRD spec을 기준으로 클러스터 안의 실제 리소스를 조정합니다.
                </Paragraph>
                <Divider />
                <Space size={8} wrap>
                  <Tag>Frontend</Tag>
                  <Tag>API</Tag>
                  <Tag>CRD</Tag>
                  <Tag>Controller</Tag>
                  <Tag>KubeVirt</Tag>
                </Space>
              </Card>
            </Col>
            <Col xs={24} lg={14}>
              <Card className="docs-timeline-card" hoverable>
                <Timeline items={architectureTimelineItems} />
              </Card>
            </Col>
          </Row>
        </section>

        <section className="docs-section">
          <Card className="docs-limit-card">
            <ControlOutlined className="docs-card-icon" />
            <Title level={2}>SSH Gateway</Title>
            <Row gutter={[20, 12]}>
              <Col xs={24} md={8}>
                <Text strong>도입한 이유</Text>
                <Paragraph>VM마다 외부 SSH 포트를 따로 열지 않기 위해 Kite Gateway가 22번 진입점을 하나로 받습니다. Gateway는 SSH username을 `spec.sshId`와 매칭해 대상 VM을 찾습니다.</Paragraph>
              </Col>
              <Col xs={24} md={8}>
                <Text strong>로그인 프록시 방식</Text>
                <Paragraph>사용자는 VM 생성 시 입력한 초기 비밀번호로 Gateway에 로그인합니다. Gateway는 이 비밀번호를 VM에 전달하지 않습니다. 인증 후 Kite 관리 SSH key로 VM sshd에 접속하고 세션만 중계합니다. 외부 사용자 public key 인증은 아직 지원하지 않습니다.</Paragraph>
              </Col>
              <Col xs={24} md={8}>
                <Text strong>host SSH 포트</Text>
                <Paragraph>설치 스크립트는 host sshd 포트를 변경하지 않습니다. 외부 VM SSH 포트는 Level 3 관리자가 Admin Settings에서 명시적으로 열고, host fallback도 host sshd 포트를 직접 입력했을 때만 활성화됩니다.</Paragraph>
              </Col>
            </Row>
          </Card>
        </section>
      </Content>
    </Layout>
  );
};
