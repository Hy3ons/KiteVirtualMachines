import React, { useState } from 'react';
import { SEO } from '../components/SEO';
import { authApi } from '../api';
import { App as AntdApp, Row, Col, Form, Input, Button, Typography, Layout } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { GlobalHeader } from '../components/GlobalHeader';
import type { SignupPayload } from '../api/types';

const { Title, Paragraph, Text } = Typography;
const { Content } = Layout;

export const SignupPage: React.FC = () => {
  const { message } = AntdApp.useApp();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);

  const onFinish = async (values: Omit<SignupPayload, 'profile_image'> & { readonly confirm: string }) => {
    try {
      setLoading(true);
      const payload = {
        username: values.username,
        email: values.email,
        password: values.password,
        profile_image: '',
      };
      await authApi.signup(payload);
      
      message.success('회원가입이 완료되었습니다! 로그인해주세요.');
      navigate('/');
    } catch (error) {
      const fallbackMessage = error instanceof Error ? error.message : '회원가입에 실패했습니다.';
      message.error(fallbackMessage);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <SEO title="회원가입 - Kite" url="/signup" />
      <GlobalHeader />
      <Content className="signup-content">
        <div className="app-main app-main--landing">
          <Row gutter={[32, 32]} align="stretch" className="signup-shell">
          {/* Left Column (60%) */}
          <Col xs={24} md={14}>
            <section className="signup-intro-panel">
            <Title level={1} style={{ fontSize: '48px', color: '#333', marginBottom: 0 }}>Kite Platform</Title>
            <Title level={3} style={{ color: '#C9B59C', marginTop: 10, marginBottom: 40 }}>
              새로운 클라우드 인프라의 시작
            </Title>
            <Paragraph style={{ fontSize: '18px', color: '#555', marginBottom: 40, maxWidth: '600px' }}>
              회원가입을 통해 당신만의 안전하고 독립적인 네임스페이스를 할당받으세요. 
              관리자의 설정에 따라 정해진 Quota 내에서 가상머신을 자유롭게 배포하고 제어할 수 있습니다.
            </Paragraph>
            </section>
          </Col>

      {/* Right Column (40%) - Signup Form */}
      <Col xs={24} md={10}>
        <section className="signup-form-panel">
        <div className="signup-form-inner">
          <div style={{ textAlign: 'center', marginBottom: 40 }}>
            <Title level={2} style={{ color: '#333' }}>Create an Account</Title>
            <Text type="secondary">Kite 플랫폼 사용을 위해 가입해주세요.</Text>
          </div>

          <Form
            name="signup_form"
            layout="vertical"
            onFinish={onFinish}
            size="large"
          >
            <Form.Item
              name="username"
              rules={[
                { required: true, message: '이름을 입력해주세요.' },
                { pattern: /^[a-zA-Z]+$/, message: '이름은 띄어쓰기 없이 영문으로만 입력해야 합니다.' }
              ]}
              extra={<span style={{ color: '#8B7355', fontSize: '12px' }}>* 이 이름은 클러스터 네임스페이스 및 리소스 식별용으로 사용됩니다.</span>}
            >
              <Input prefix={<UserOutlined style={{ color: '#bfbfbf' }} />} placeholder="이름 (영문 입력, 예: hyeonseok)" />
            </Form.Item>

            <Form.Item
              name="email"
              rules={[
                { required: true, message: '이메일을 입력해주세요.' },
                { type: 'email', message: '유효한 이메일 형식이 아닙니다.' }
              ]}
              extra={<span style={{ color: '#d9363e', fontSize: '12px' }}>* 이메일은 로그인 아이디로 사용되며, 가입 후 절대 변경할 수 없습니다.</span>}
            >
              <Input prefix={<MailOutlined style={{ color: '#bfbfbf' }} />} placeholder="kite@example.com" />
            </Form.Item>

            <Form.Item
              name="password"
              rules={[{ required: true, message: '비밀번호를 입력해주세요.' }]}
              hasFeedback
            >
              <Input.Password prefix={<LockOutlined style={{ color: '#bfbfbf' }} />} placeholder="Password" />
            </Form.Item>

            <Form.Item
              name="confirm"
              dependencies={['password']}
              hasFeedback
              rules={[
                { required: true, message: '비밀번호 확인을 입력해주세요.' },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('password') === value) {
                      return Promise.resolve();
                    }
                    return Promise.reject(new Error('비밀번호가 일치하지 않습니다.'));
                  },
                }),
              ]}
            >
              <Input.Password prefix={<LockOutlined style={{ color: '#bfbfbf' }} />} placeholder="Confirm Password" />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} block style={{ height: '48px', fontSize: '16px' }}>
                회원가입 완료
              </Button>
            </Form.Item>
          </Form>

          <div style={{ textAlign: 'center', marginTop: 20 }}>
            <Text type="secondary">이미 계정이 있으신가요? <span style={{ color: '#C9B59C', cursor: 'pointer' }} onClick={() => navigate('/')}>로그인하러 가기</span></Text>
          </div>
        </div>
        </section>
      </Col>
        </Row>
        </div>
      </Content>
    </Layout>
  );
};
