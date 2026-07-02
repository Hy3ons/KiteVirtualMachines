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
    rules: ['전체 VM 제어, 사용자 삭제, 시스템 설정을 허용합니다.', '운영 위험이 큰 작업은 UI에서 명확히 구분합니다.'],
  },
];

const architectureTimelineItems: TimelineProps['items'] = [
  {
    color: '#8B7355',
    content: (
      <div>
        <Text strong>사용자가 화면에서 요청</Text>
        <Paragraph className="docs-timeline-copy">사용자는 Dashboard에서 로그인, VM 생성, 전원 변경 같은 작업을 요청합니다. 화면은 API 응답과 VM 상태를 보여줍니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#C9B59C',
    content: (
      <div>
        <Text strong>API가 권한과 입력 확인</Text>
        <Paragraph className="docs-timeline-copy">`kite-api`는 세션과 access level을 확인하고, 허용된 요청만 Kubernetes API 서버에 기록합니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#7A9C74',
    content: (
      <div>
        <Text strong>CRD가 원하는 상태 보관</Text>
        <Paragraph className="docs-timeline-copy">`KiteUser`와 `KiteVirtualMachine` CRD의 spec에는 사용자가 원하는 계정과 VM 상태가 저장됩니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#D4A373',
    content: (
      <div>
        <Text strong>Controller가 실제 리소스 조정</Text>
        <Paragraph className="docs-timeline-copy">`kite-controller`는 CRD spec을 읽고 KubeVirt VM, DataVolume, 서비스 같은 실제 Kubernetes 리소스를 맞춥니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#8B7355',
    content: (
      <div>
        <Text strong>상태가 다시 화면으로 반영</Text>
        <Paragraph className="docs-timeline-copy">controller는 관측한 상태와 실패 이유를 CRD status에 기록하고, Frontend는 그 값을 사용자에게 보여줍니다.</Paragraph>
      </div>
    ),
  },
];

export const KiteDocsPage: React.FC = () => {
  const navigate = useNavigate();

  return (
    <Layout style={{ minHeight: '100vh', backgroundColor: '#F9F8F6' }}>
      <SEO
        title="Kite Docs - 권한 모델과 아키텍처"
        description="Kite 권한 레벨, VM 생성 제한, Frontend, API, CRD, controller, KubeVirt 흐름과 현재 한계를 정리한 문서입니다."
        url="/kite-docs"
      />
      <GlobalHeader />

      <Content className="app-main app-main--wide docs-content">
        <section className="docs-hero">
          <Space size={8} wrap className="docs-kicker">
            <Tag>Access Control</Tag>
            <Tag>Architecture</Tag>
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
            <Title level={2}>현재 한계와 운영 주의</Title>
            <Row gutter={[20, 12]}>
              <Col xs={24} md={8}>
                <Text strong>비밀번호 변경</Text>
                <Paragraph>VM 생성 시 정한 초기 비밀번호만 Kite가 전달합니다. 생성 후 변경/복구는 guest OS 안에서 처리해야 합니다.</Paragraph>
              </Col>
              <Col xs={24} md={8}>
                <Text strong>상태 반영</Text>
                <Paragraph>VM 생성과 전원 변경은 controller reconcile을 거쳐 반영되므로 화면 상태가 잠시 처리 중으로 보일 수 있습니다.</Paragraph>
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
