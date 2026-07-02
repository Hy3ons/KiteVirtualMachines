import React from 'react';
import { SEO } from '../components/SEO';
import { GlobalHeader } from '../components/GlobalHeader';
import type { TimelineProps } from 'antd';
import { Alert, Button, Card, Col, Divider, Layout, Row, Space, Tag, Timeline, Typography } from 'antd';
import {
  ApartmentOutlined,
  BookOutlined,
  CloudServerOutlined,
  ControlOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
  UserSwitchOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

const { Content } = Layout;
const { Title, Paragraph, Text } = Typography;

type AccessLevelDoc = {
  readonly level: string;
  readonly title: string;
  readonly tone: string;
  readonly summary: string;
  readonly rules: readonly string[];
};

const accessLevelDocs: readonly AccessLevelDoc[] = [
  {
    level: '0',
    title: 'No VM Create',
    tone: '권한 요청 단계',
    summary: '로그인은 허용하지만 VM 생성과 제어는 제한합니다.',
    rules: ['Dashboard에 관리자 연락처를 표시합니다.', 'API 직접 호출로 VM 생성/수정/전원/삭제를 시도하면 403을 반환합니다.'],
  },
  {
    level: '1',
    title: 'Fixed Self-service',
    tone: '일반 사용자',
    summary: '자기 namespace 안에서 고정 스펙 VM만 생성하고 관리할 수 있습니다.',
    rules: ['CPU 2, Memory 4Gi, Disk 25Gi로 고정합니다.', '직접 생성할 수 있는 VM은 최대 2개로 제한합니다.'],
  },
  {
    level: '2',
    title: 'Power User',
    tone: '팀 운영자',
    summary: '자기 VM은 자유 스펙으로 만들 수 있고, 사용자 권한은 0/1 범위에서만 조정할 수 있습니다.',
    rules: ['다른 사용자의 VM 전원/삭제/설정 변경은 허용하지 않습니다.', '관리 화면에는 0과 1 사이의 권한 변경만 노출합니다.'],
  },
  {
    level: '3',
    title: 'Full Admin',
    tone: '전체 관리자',
    summary: 'Kite의 모든 관리 작업을 수행할 수 있습니다.',
    rules: ['전체 VM 제어, 사용자 삭제, 시스템 설정, VM Offer 생성/삭제를 허용합니다.', '운영 위험이 큰 작업은 UI에서 명확히 구분합니다.'],
  },
];

const offerTimelineItems: TimelineProps['items'] = [
  {
    color: '#8B7355',
    content: (
      <div>
        <Text strong>관리자가 Offer 생성</Text>
        <Paragraph className="docs-timeline-copy">관리자는 대상 사용자 namespace에 CPU, memory, disk, image, 만료 시간을 가진 `KiteVirtualMachineOffer`를 생성합니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#C9B59C',
    content: (
      <div>
        <Text strong>사용자가 Claim</Text>
        <Paragraph className="docs-timeline-copy">사용자는 자기 namespace의 Offer만 확인하고, VM name, domain prefix, SSH ID, 초기 비밀번호를 입력합니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#7A9C74',
    content: (
      <div>
        <Text strong>실제 VM으로 전환</Text>
        <Paragraph className="docs-timeline-copy">Claim이 성공하면 Kite는 Offer 스펙과 사용자 입력을 합쳐 `KiteVirtualMachine`을 만들고 Offer를 정리합니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#D4A373',
    content: (
      <div>
        <Text strong>24시간 TTL 정리</Text>
        <Paragraph className="docs-timeline-copy">사용되지 않은 Offer는 controller cleanup으로 삭제해 불필요한 예약이 쌓이지 않도록 관리합니다.</Paragraph>
      </div>
    ),
  },
];

export const KiteDocsPage: React.FC = () => {
  const navigate = useNavigate();

  return (
    <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
      <SEO
        title="Kite Docs - 권한 모델과 VM Offer 설계"
        description="Kite 권한 레벨, VM 생성 제한, 관리자 VM Offer 설계와 현재 한계를 정리한 문서입니다."
        url="/kite-docs"
      />
      <GlobalHeader />

      <Content className="app-main app-main--wide docs-content">
        <section className="docs-hero">
          <Space size={8} wrap className="docs-kicker">
            <Tag>Access Control</Tag>
            <Tag>VM Offer</Tag>
            <Tag>KubeVirt</Tag>
          </Space>
          <Title level={1} className="docs-title">Kite Docs</Title>
          <Paragraph className="docs-subtitle">
            Kite의 권한 모델, VM 생성 제한, 관리자가 사용자에게 VM 스펙을 배정하는 Offer 흐름을 한 곳에 정리합니다.
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
                  일반 사용자는 직접 VM을 만들 수 있지만 스펙은 고정됩니다. UI 안내뿐 아니라 백엔드에서도 같은 정책을 강제합니다.
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
                  Level 2는 자기 VM을 자유 스펙으로 만들 수 있지만 전체 관리자 권한은 아닙니다. 사용자 권한은 0과 1 사이에서만 변경할 수 있습니다.
                </Paragraph>
                <Alert type="info" showIcon title="Level 2는 다른 사용자의 VM 전원, 삭제, 설정 변경을 허용하지 않습니다." />
              </Card>
            </Col>
            <Col xs={24} lg={8}>
              <Card className="docs-policy-card" hoverable>
                <LockOutlined className="docs-card-icon" />
                <Title level={3}>관리자 연락처</Title>
                <Paragraph>
                  Level 0 사용자는 VM 생성 버튼 대신 관리자 연락 안내를 봅니다. 연락처는 `kite-runtime-config`에서 관리합니다.
                </Paragraph>
                <Alert type="warning" showIcon title="adminContact는 email, Slack, 전화번호, URL 등 자유 텍스트로 설정할 수 있습니다." />
              </Card>
            </Col>
          </Row>
        </section>

        <section className="docs-section docs-offer-section">
          <div className="docs-section-heading">
            <Title level={2}>VM Offer 설계</Title>
            <Text type="secondary">관리자가 스펙을 먼저 정하고, 사용자가 나머지 정보를 입력해 자기 VM으로 claim하는 흐름입니다.</Text>
          </div>
          <Row gutter={[24, 24]}>
            <Col xs={24} lg={10}>
              <Card className="docs-offer-card" hoverable>
                <ApartmentOutlined className="docs-card-icon" />
                <Title level={3}>왜 별도 CRD인가</Title>
                <Paragraph>
                  미완성 VM을 `KiteVirtualMachine`에 넣으면 controller가 provisioning 대상인지, 예약값인지 계속 구분해야 합니다.
                  Kite는 Offer를 별도 CRD로 분리해 claim, 만료, 권한 감사 흐름을 단순하게 유지합니다.
                </Paragraph>
                <Divider />
                <Space size={8} wrap>
                  <Tag>Namespaced</Tag>
                  <Tag>24h TTL</Tag>
                  <Tag>Claim once</Tag>
                  <Tag>Admin only</Tag>
                </Space>
              </Card>
            </Col>
            <Col xs={24} lg={14}>
              <Card className="docs-timeline-card" hoverable>
                <Timeline items={offerTimelineItems} />
              </Card>
            </Col>
          </Row>
        </section>

        <section className="docs-section">
          <Card className="docs-limit-card">
            <ControlOutlined className="docs-card-icon" />
            <Title level={2}>현재 한계와 운영 주의</Title>
            <Row gutter={[20, 12]}>
              <Col xs={24} md={8}>
                <Text strong>비밀번호 변경</Text>
                <Paragraph>VM 생성 시 정한 초기 비밀번호만 Kite가 전달합니다. 생성 후 변경/복구는 guest OS 안에서 처리해야 합니다.</Paragraph>
              </Col>
              <Col xs={24} md={8}>
                <Text strong>Offer 우회 정책</Text>
                <Paragraph>관리자 Offer는 명시적 배정이므로 Level 1의 고정 스펙 제한을 우회할 수 있습니다.</Paragraph>
              </Col>
              <Col xs={24} md={8}>
                <Text strong>최종 권한 판단</Text>
                <Paragraph>Frontend는 안내와 UX를 담당하고, 실제 권한 차단은 반드시 API 서버에서 같은 규칙으로 강제합니다.</Paragraph>
              </Col>
            </Row>
          </Card>
        </section>
      </Content>
    </Layout>
  );
};
