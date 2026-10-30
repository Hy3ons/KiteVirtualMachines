import { useEffect } from 'react';
import { Form } from 'antd';
import { DemoPublicHeader } from './DemoPublicHeader';
import { LoginForm } from '../LoginForm';
import type { LoginFormValues } from '../LoginForm';
import type { DemoSnapshot } from './demoStory';
import { demoIgnore } from './demoData';

// DemoLogin uses the real login fields with a local form, never the authentication adapter.
export function DemoLogin({ snapshot }: { readonly snapshot: DemoSnapshot }) {
  const [form] = Form.useForm<LoginFormValues>();
  useEffect(() => { form.setFieldsValue({ username: snapshot.loginEmail, password: snapshot.loginPassword }); }, [form, snapshot.loginEmail, snapshot.loginPassword]);
  return <div className="demo-login public-page">
    <DemoPublicHeader />
    <div className="marketing-login-page"><div className="marketing-auth-panel">
      <div className="marketing-form-heading"><h2>Kite에 로그인</h2></div>
      <LoginForm name="demo_login" form={form} loading={snapshot.busy} onFinish={demoIgnore} />
      <p className="marketing-form-footer">아직 계정이 없으신가요? <span>회원가입</span></p>
    </div></div>
  </div>;
}
