import React, { useState, useEffect } from 'react';
import { SEO } from '../components/SEO';
import { authApi } from '../api';
import { App as AntdApp, Row, Col, Form, Input, Button, Typography, Layout, Space, Tag } from 'antd';
import { UserOutlined, LockOutlined, CloudServerOutlined, SecurityScanOutlined, BuildOutlined, GithubOutlined, DashboardOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { useLogout } from '../hooks/useLogout';
import { useNavigate, useLocation } from 'react-router-dom';
import { GlobalHeader } from '../components/GlobalHeader';
import type { LoginCredentials } from '../api/types';

const { Title, Paragraph, Text } = Typography;
const { Content } = Layout;

export const LandingPage: React.FC = () => {
  const { message, notification } = AntdApp.useApp();
  const { login, authenticated, username, accessLevel, profileImage } = useAuthStore();
  const logout = useLogout();
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
      
      const { user } = data;
      login(user.access_level, user.username, user.namespace, user.profile_image || '');
      
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
                <div className="landing-product-heading">
                  <Title level={1} className="landing-title">Kite</Title>
                  <Text className="landing-title-caption">Opensource Private Cloud</Text>
                </div>
                <Paragraph className="landing-subtitle">
                  Kubernetes 위에서 돌아가는 KubeVirt를 응용해 더욱 추상화된 VM 상태 관리를 제공합니다.
                  간단한 웹 UI로 VM을 대여하고, 서버 호스트의 sshd를 대신하는 kite-sshgateway로 접속 흐름을 최적화합니다.
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
                {authenticated ? (
                  <div className="landing-session-card">
                    <div className="landing-profile-mark">
                      {profileImage ? (
                        <img src={profileImage} alt="Profile" className="landing-profile-image" />
                      ) : (
                        <UserOutlined className="landing-profile-icon" />
                      )}
                    </div>
                    <Title level={3} className="landing-login-title">{username}님 환영합니다!</Title>
                    <Text type="secondary" className="landing-login-caption">현재 로그인된 상태입니다.</Text>

                    <Button
                      type="primary"
                      size="large"
                      block
                      icon={<DashboardOutlined />}
                      onClick={() => navigate('/dashboard')}
                      className="landing-primary-action"
                    >
                      My VMs로 이동하기
                    </Button>
                    {accessLevel !== null && accessLevel >= 2 && (
                      <Button block onClick={() => navigate('/admin/dashboard')} className="landing-outline-action">
                        Admin Console
                      </Button>
                    )}
                    <Button type="text" onClick={logout} block className="landing-text-action">
                      로그아웃
                    </Button>
                  </div>
                ) : (
                  <>
                    <div className="landing-login-intro">
                      <div className="landing-orbit-mark">
                        <img src="/favicon.png" alt="Kite logo" />
                      </div>
                      <Title level={2} className="landing-login-title">Welcome to Kite</Title>
                      <Text type="secondary" className="landing-login-caption">로그인 후 VM 대여 대시보드로 이동합니다.</Text>
                    </div>

                    <Form
                      name="login_form"
                      layout="vertical"
                      onFinish={onFinish}
                      size="large"
                      className="landing-login-form"
                    >
                      <Form.Item
                        name="username"
                        rules={[{ required: true, message: '아이디를 입력해주세요.' }]}
                      >
                        <Input prefix={<UserOutlined className="landing-input-icon" />} placeholder="Email" />
                      </Form.Item>

                      <Form.Item
                        name="password"
                        rules={[{ required: true, message: '비밀번호를 입력해주세요.' }]}
                      >
                        <Input.Password prefix={<LockOutlined className="landing-input-icon" />} placeholder="Password" />
                      </Form.Item>

                      <Form.Item>
                        <Button type="primary" htmlType="submit" block loading={loading} className="landing-primary-action">
                          Log in
                        </Button>
                      </Form.Item>

                      <div className="landing-signup-row">
                        <Text type="secondary">계정이 없으신가요? </Text>
                        <Button type="link" onClick={() => navigate('/signup')} className="landing-signup-link">회원가입</Button>
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
