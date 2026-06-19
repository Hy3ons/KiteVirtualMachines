import React, { useState, useEffect } from 'react';
import { SEO } from '../components/SEO';
import { authApi } from '../api';
import { App as AntdApp, Row, Col, Form, Input, Button, Typography, Layout } from 'antd';
import { UserOutlined, LockOutlined, CloudServerOutlined, SecurityScanOutlined, BuildOutlined, GithubOutlined } from '@ant-design/icons';
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
      if (user.access_level >= 2) {
        navigate('/admin/dashboard');
      } else {
        navigate('/dashboard');
      }
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
      <Content>
        <Row style={{ margin: 0, width: '100%', minHeight: 'calc(100vh - 65px)', backgroundColor: '#F9F8F6' }}>
          {/* Left Column (60%) - Project Description */}
      <Col xs={0} md={14} style={{ backgroundColor: '#EFE9E3', padding: '60px', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
        <img src="/favicon.png" alt="Kite Architecture Logo" style={{ width: '100%', maxWidth: '300px', objectFit: 'contain', mixBlendMode: 'multiply', margin: '-40px 0 -40px -40px' }} />
        <Title level={1} style={{ fontSize: '48px', color: '#333', marginBottom: 0, zIndex: 1 }}>Kite Platform</Title>
        <Title level={3} style={{ color: '#C9B59C', marginTop: 10, marginBottom: 40, zIndex: 1 }}>
          Kubernetes 기반의 안전한 가상머신 프로비저닝
        </Title>

        <Paragraph style={{ fontSize: '18px', color: '#555', marginBottom: 40, maxWidth: '600px' }}>
          Kite는 개발자와 관리자 모두에게 직관적인 웹 대시보드를 제공하여, 네임스페이스 격리 수준의 안전한 가상머신 리소스를 단 몇 초 만에 배포하고 관리할 수 있게 해줍니다.
        </Paragraph>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '30px' }}>
          <div style={{ display: 'flex', alignItems: 'flex-start' }}>
            <CloudServerOutlined style={{ fontSize: '24px', color: '#C9B59C', marginRight: '16px', marginTop: '4px' }} />
            <div>
              <Text strong style={{ fontSize: '16px' }}>Instant Provisioning</Text>
              <Paragraph style={{ margin: 0, color: '#666' }}>필요한 스펙의 가상머신을 즉시 할당받고 제어하세요.</Paragraph>
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'flex-start' }}>
            <SecurityScanOutlined style={{ fontSize: '24px', color: '#C9B59C', marginRight: '16px', marginTop: '4px' }} />
            <div>
              <Text strong style={{ fontSize: '16px' }}>Perfect Isolation</Text>
              <Paragraph style={{ margin: 0, color: '#666' }}>유저별 독립된 네임스페이스와 리소스 Quota로 시스템을 안전하게 보호합니다.</Paragraph>
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'flex-start' }}>
            <BuildOutlined style={{ fontSize: '24px', color: '#C9B59C', marginRight: '16px', marginTop: '4px' }} />
            <div>
              <Text strong style={{ fontSize: '16px' }}>Global Architecture</Text>
              <Paragraph style={{ margin: 0, color: '#666' }}>Ingress와 VM별 SSH 계정이 자동 연동되어 배포 즉시 서비스 접속이 가능합니다.</Paragraph>
            </div>
          </div>
        </div>

        <div style={{ marginTop: '60px' }}>
          <Button 
            size="large" 
            icon={<GithubOutlined />} 
            href="https://github.com/Hy3ons/KiteVirtualMachines" 
            target="_blank"
            style={{ borderRadius: 4, height: '48px', fontSize: '16px', display: 'flex', alignItems: 'center', width: 'fit-content', background: '#333', color: '#fff', border: 'none' }}
          >
            View on GitHub
          </Button>
        </div>
      </Col>

      {/* Right Column (40%) - Login Form or Welcome Back */}
      <Col xs={24} md={10} style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '40px' }}>
        <div style={{ width: '100%', maxWidth: '360px' }}>
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
                onClick={() => navigate(accessLevel && accessLevel >= 2 ? '/admin/dashboard' : '/dashboard')}
                style={{ backgroundColor: '#8B7355', borderColor: '#8B7355', height: '48px', marginBottom: 16 }}
              >
                대시보드로 이동하기
              </Button>
              <Button type="text" onClick={logout} block>
                로그아웃
              </Button>
            </div>
          ) : (
            <>
              <div style={{ textAlign: 'center', marginBottom: 40 }}>
                <Title level={2} style={{ color: '#333' }}>Welcome to Kite</Title>
                <Text type="secondary">관리자 또는 일반 계정으로 로그인하세요.</Text>
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
        </div>
      </Col>
        </Row>
      </Content>
    </Layout>
  );
};
