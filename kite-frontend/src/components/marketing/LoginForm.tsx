import { Button, Form, Input } from 'antd';
import type { FormInstance } from 'antd';
import { ArrowRightOutlined } from '@ant-design/icons';

export type LoginFormValues = { readonly username: string; readonly password: string };
type LoginFormProps = {
  readonly loading: boolean;
  readonly onFinish: (values: LoginFormValues) => void;
  readonly form?: FormInstance<LoginFormValues>;
  readonly name?: string;
};

// LoginForm shares the labelled login fields between real authentication and the inert product demo.
// onFinish owns submission; this presentation component never calls authentication APIs itself.
export function LoginForm({ loading, onFinish, form, name = 'login_form' }: LoginFormProps) {
  return (
    <Form<LoginFormValues> form={form} name={name} layout="vertical" onFinish={onFinish} size="large" requiredMark={false}>
      <Form.Item name="username" label="이메일" rules={[{ required: true, message: '이메일을 입력해주세요.' }]}>
        <Input type="email" autoComplete="username" placeholder="you@example.com" />
      </Form.Item>
      <Form.Item name="password" label="비밀번호" rules={[{ required: true, message: '비밀번호를 입력해주세요.' }]}>
        <Input.Password autoComplete="current-password" placeholder="비밀번호를 입력하세요" />
      </Form.Item>
      <Form.Item><Button type="primary" htmlType="submit" block loading={loading} className="marketing-submit">로그인 <ArrowRightOutlined /></Button></Form.Item>
    </Form>
  );
}
