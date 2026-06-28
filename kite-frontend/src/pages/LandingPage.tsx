import React, { useState, useEffect } from 'react';
import { SEO } from '../components/SEO';
import { authApi } from '../api';
import { App as AntdApp, Row, Col, Form, Input, Button, Typography, Layout, Space, Tag } from 'antd';
import { UserOutlined, LockOutlined, CloudServerOutlined, SecurityScanOutlined, BuildOutlined, GithubOutlined, DashboardOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { useNavigate, useLocation } from 'react-router-dom';
import { GlobalHeader } from '../components/GlobalHeader';
import type { LoginCredentials } from '../api/types';

const { Title, Paragraph, Text } = Typography;
const { Content } = Layout;

export const LandingPage: React.FC = () => {
  const { message, notification } = AntdApp.useApp();
  const { login, token, username, accessLevel, profileImage, logout } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    // 환경변수를 통해 Production 환경인지 확인 (Vite 기준 import.meta.env.PROD 사용)
    const isProd = import.meta.env.PROD;
    
    // if (isProd && location.state?.requireLogin) { // <-- 유저가 원한 조건이지만 쾌적한 디버깅을 위해 일단 조건 없이도 작동하게 둘수도 있습니다. (isProd 명시함)
    if (isProd && location.state?.requireLogin) {
      notification.warning({
        message: '로그인이 필요한 서비스입니다.',
        description: `접근하시려는 페이지(${location.state.path})는 로그인 후 이용하실 수 있습니다.`,
        placement: 'top',
        duration: 4,
        style: { borderRadius: 4, border: '1px solid #D9CFC7' }
      });
      // 경고를 띄운 후에는 state를 초기화하여 새로고침 시 다시 뜨지 않도록 함
      window.history.replaceState({}, document.title);
    }
  }, [location, notification]);

  const onFinish = async (values: LoginCredentials & { readonly username: string }) => {
    try {
      setLoading(true);
      const data = await authApi.login({ email: values.username, password: values.password });
      
      const { accessToken, user } = data;
      login(accessToken, user.access_level, user.username, user.namespace, user.profile_image || '');
      
      message.success('로그인 되었습니다.');
      navigate('/dashboard');
    } catch (error) {
      const fallbackMessage = error instanceof Error ? error.message : '로그인에 실패했습니다.';
      message.error(fallbackMessage);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <SEO title="Kite - Kubernetes Virtual Machines" url="/" />
      <GlobalHeader />
      <Content className="landing-content">
        <div className="app-main app-main--landing">
          <Row gutter={[32, 32]} align="stretch" className="landing-grid">
            <Col xs={24} lg={14}>
              <section className="landing-hero-panel">
                <Space size={8} wrap className="landing-kicker">
                  <Tag>Kubernetes</Tag>
                  <Tag>KubeVirt</Tag>
                  <Tag>Namespace isolated</Tag>
                </Space>
                <img src="/favicon.png" alt="Kite logo" className="landing-logo-mark" />
                <Title level={1} className="landing-title">Kite Platform</Title>
                <Paragraph className="landing-subtitle">
                  팀원이 필요한 VM을 직접 대여하고, 관리자는 같은 화면에서 전체 상태를 조용하게 통제합니다.
                </Paragraph>

                <div className="landing-feature-list">
                  <div className="landing-feature">
                    <CloudServerOutlined />
                    <div>
                      <Text strong>Self-service VM rental</Text>
                      <Paragraph>권한에 맞는 CPU, RAM, disk 정책으로 VM을 생성하고 시작합니다.</Paragraph>
                    </div>
                  </div>
                  <div className="landing-feature">
                    <SecurityScanOutlined />
                    <div>
                      <Text strong>Scoped by namespace</Text>
                      <Paragraph>사용자별 네임스페이스와 quota 기준으로 리소스 경계를 유지합니다.</Paragraph>
                    </div>
                  </div>
                  <div className="landing-feature">
                    <BuildOutlined />
                    <div>
                      <Text strong>Console and SSH ready</Text>
                      <Paragraph>웹 콘솔, VM 도메인, SSH gateway 흐름을 한 대시보드에서 다룹니다.</Paragraph>
                    </div>
                  </div>
                </div>

                <Button
                  size="large"
                  icon={<GithubOutlined />}
                  href="https://github.com/Hy3ons/KiteVirtualMachines"
                  target="_blank"
                  className="landing-secondary-action"
                >
                  Repository
                </Button>
              </section>
            </Col>

            <Col xs={24} lg={10}>
              <section className="landing-login-panel">
                {token ? (
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ marginBottom: 24 }}>
                      {profileImage ? (
                        <img src={profileImage} alt="Profile" style={{ width: 80, height: 80, borderRadius: '50%', objectFit: 'cover' }} />
                      ) : (
                        <div style={{ width: 80, height: 80, borderRadius: '50%', backgroundColor: '#f0f0f0', display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}>
                          <UserOutlined style={{ fontSize: 32, color: '#bfbfbf' }} />
                        </div>
                      )}
                    </div>
                    <Title level={3} style={{ color: '#333' }}>{username}님 환영합니다!</Title>
                    <Text type="secondary" style={{ display: 'block', marginBottom: 32 }}>현재 로그인된 상태입니다.</Text>

                    <Button
                      type="primary"
                      size="large"
                      block
                      icon={<DashboardOutlined />}
                      onClick={() => navigate('/dashboard')}
                      style={{ backgroundColor: '#8B7355', borderColor: '#8B7355', height: '48px', marginBottom: 16 }}
                    >
                      My VMs로 이동하기
                    </Button>
                    {accessLevel !== null && accessLevel >= 2 && (
                      <Button block onClick={() => navigate('/admin/dashboard')} style={{ height: '44px', marginBottom: 12 }}>
                        Admin Console
                      </Button>
                    )}
                    <Button type="text" onClick={logout} block>
                      로그아웃
                    </Button>
                  </div>
                ) : (
                  <>
                    <div style={{ textAlign: 'center', marginBottom: 40 }}>
                      <Title level={2} style={{ color: '#333' }}>Welcome to Kite</Title>
                      <Text type="secondary">로그인 후 VM 대여 대시보드로 이동합니다.</Text>
                    </div>

                    <Form
                      name="login_form"
                      layout="vertical"
                      onFinish={onFinish}
                      size="large"
                    >
                      <Form.Item
                        name="username"
                        rules={[{ required: true, message: '아이디를 입력해주세요.' }]}
                      >
                        <Input prefix={<UserOutlined style={{ color: '#bfbfbf' }} />} placeholder="Email" />
                      </Form.Item>

                      <Form.Item
                        name="password"
                        rules={[{ required: true, message: '비밀번호를 입력해주세요.' }]}
                      >
                        <Input.Password prefix={<LockOutlined style={{ color: '#bfbfbf' }} />} placeholder="Password" />
                      </Form.Item>

                      <Form.Item>
                        <Button type="primary" htmlType="submit" block loading={loading} style={{ backgroundColor: '#8B7355', borderColor: '#8B7355', height: '48px', fontSize: '16px' }}>
                          Log in
                        </Button>
                      </Form.Item>

                      <div style={{ textAlign: 'center', marginTop: 16 }}>
                        <Text type="secondary">계정이 없으신가요? </Text>
                        <Button type="link" onClick={() => navigate('/signup')} style={{ padding: 0, color: '#8B7355' }}>회원가입</Button>
                      </div>
                    </Form>
                  </>
                )}
              </section>
            </Col>
          </Row>
        </div>
      </Content>
    </Layout>
  );
};
