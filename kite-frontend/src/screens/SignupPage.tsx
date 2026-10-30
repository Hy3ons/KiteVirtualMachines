'use client';

import { useState } from 'react';
import Link from 'next/link';
import { App, Button, Form, Input } from 'antd';
import { ArrowRightOutlined, CheckOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { authApi } from '../api';
import { GlobalHeader } from '../components/GlobalHeader';
import type { SignupPayload } from '../api/types';

type SignupFormValues = Omit<SignupPayload, 'profile_image'> & { readonly confirm: string };

// SignupPage presents account registration while preserving Go validation and cookie ownership.
// Successful registration returns the visitor to the dedicated login page.
export function SignupPage() {
  const { message } = App.useApp();
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  // submitSignup sends validated account fields to Go; confirm is only used by the form.
  // The empty profile image preserves the existing registration payload contract.
  const submitSignup = async (values: SignupFormValues) => {
    try {
      setLoading(true);
      await authApi.signup({
        username: values.username,
        email: values.email,
        password: values.password,
        profile_image: '',
      });
      message.success('회원가입이 완료되었습니다! 로그인해주세요.');
      router.push('/login');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '회원가입에 실패했습니다.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="public-page">
      <GlobalHeader />
      <main id="main-content" className="marketing-signup">
        <section className="marketing-signup-story" aria-labelledby="signup-title">
          <div>
            <p className="marketing-eyebrow">Welcome to your workspace</p>
            <h1 id="signup-title">작업의 시작,<br />나만의 Kite.</h1>
            <p className="marketing-lead">계정을 만들고 내 작업 공간에서<br className="marketing-desktop-break" /> 가상머신을 관리하세요.</p>
          </div>
          <div className="marketing-signup-note">
            <p className="marketing-eyebrow">시작하기 전에</p>
            <ul>
              <li><CheckOutlined /><span>영문 이름은 계정 식별에 사용합니다.</span></li>
              <li><CheckOutlined /><span>이메일은 로그인 아이디이며 가입 후 변경할 수 없습니다.</span></li>
              <li><CheckOutlined /><span>VM 생성과 사용 범위는 관리자가 부여한 권한에 따릅니다.</span></li>
            </ul>
            <Link href="/kite-docs#access-levels" className="marketing-text-link">계정 권한 알아보기 <ArrowRightOutlined /></Link>
          </div>
        </section>
        <section className="marketing-signup-form" aria-labelledby="signup-form-title">
          <div className="marketing-form-heading"><h2 id="signup-form-title">계정 만들기</h2><p>몇 가지 정보로 Kite를 시작하세요.</p></div>
          <Form<SignupFormValues> name="signup_form" layout="vertical" onFinish={submitSignup} size="large" requiredMark={false}>
            <Form.Item name="username" label="이름" rules={[
              { required: true, message: '이름을 입력해주세요.' },
              { pattern: /^[a-zA-Z]+$/, message: '이름은 띄어쓰기 없이 영문으로만 입력해야 합니다.' },
            ]} extra="띄어쓰기 없이 영문으로 입력하세요.">
              <Input autoComplete="username" placeholder="예: hyeonseok" />
            </Form.Item>
            <Form.Item name="email" label="이메일" rules={[
              { required: true, message: '이메일을 입력해주세요.' },
              { type: 'email', message: '유효한 이메일 형식이 아닙니다.' },
            ]} extra="가입 후 변경할 수 없는 로그인 아이디입니다.">
              <Input type="email" autoComplete="email" placeholder="you@example.com" />
            </Form.Item>
            <Form.Item name="password" label="비밀번호" rules={[{ required: true, message: '비밀번호를 입력해주세요.' }]} hasFeedback>
              <Input.Password autoComplete="new-password" placeholder="비밀번호를 입력하세요" />
            </Form.Item>
            <Form.Item name="confirm" label="비밀번호 확인" dependencies={['password']} hasFeedback rules={[
              { required: true, message: '비밀번호 확인을 입력해주세요.' },
              ({ getFieldValue }) => ({
                // The confirmation validator compares user input with the current password field.
                validator(_, value: unknown) {
                  return !value || getFieldValue('password') === value
                    ? Promise.resolve()
                    : Promise.reject(new Error('비밀번호가 일치하지 않습니다.'));
                },
              }),
            ]}>
              <Input.Password autoComplete="new-password" placeholder="비밀번호를 다시 입력하세요" />
            </Form.Item>
            <Form.Item><Button type="primary" htmlType="submit" loading={loading} block className="marketing-submit">회원가입 <ArrowRightOutlined /></Button></Form.Item>
          </Form>
          <p className="marketing-form-footer">이미 계정이 있으신가요? <Link href="/login">로그인</Link></p>
        </section>
      </main>
    </div>
  );
}
