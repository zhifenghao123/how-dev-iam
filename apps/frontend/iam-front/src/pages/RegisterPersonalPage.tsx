import { useState } from 'react';
import { Card, Form, Input, Button, Space, Typography, message } from 'antd';
import { useNavigate, Link } from 'react-router-dom';
import { apiSendVerifyCode, apiRegisterPersonal } from '@/api';
import { useAuthStore } from '@/stores/authStore';

const { Title, Text } = Typography;

// 个人注册页：邮箱 + 验证码 + 密码 + 昵称。
export default function RegisterPersonalPage() {
  const nav = useNavigate();
  const setUser = useAuthStore((s) => s.setUser);
  const [form] = Form.useForm();
  const [sendingCode, setSendingCode] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [countdown, setCountdown] = useState(0);

  const sendCode = async () => {
    const email = form.getFieldValue('email');
    if (!email) {
      message.error('请先填写邮箱');
      return;
    }
    setSendingCode(true);
    try {
      const r = await apiSendVerifyCode({
        channel: 'email',
        target: email,
        scene: 'register',
      });
      message.success(
        r.debug_code
          ? `验证码已发送（开发模式：${r.debug_code}）`
          : '验证码已发送到邮箱',
      );
      let n = 60;
      setCountdown(n);
      const t = setInterval(() => {
        n -= 1;
        setCountdown(n);
        if (n <= 0) clearInterval(t);
      }, 1000);
    } catch {
      // interceptor 已 toast
    } finally {
      setSendingCode(false);
    }
  };

  const onSubmit = async (v: any) => {
    setSubmitting(true);
    try {
      const r = await apiRegisterPersonal({
        email: v.email,
        code: v.code,
        password: v.password,
        nickname: v.nickname,
      });
      setUser(r.user_id, v.nickname || v.email);
      message.success('注册成功');
      nav('/dashboard');
    } catch {
      // toast handled
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f2f5',
      }}
    >
      <Card style={{ width: 460 }}>
        <Title level={3} style={{ textAlign: 'center' }}>
          个人注册
        </Title>
        <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
          注册成功后会自动为你创建一个个人账户。
        </Text>
        <Form form={form} layout="vertical" onFinish={onSubmit}>
          <Form.Item
            name="email"
            label="邮箱"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          >
            <Input placeholder="you@example.com" />
          </Form.Item>
          <Form.Item name="code" label="验证码" rules={[{ required: true }]}>
            <Space.Compact style={{ width: '100%' }}>
              <Input placeholder="6 位验证码" />
              <Button
                onClick={sendCode}
                loading={sendingCode}
                disabled={countdown > 0}
              >
                {countdown > 0 ? `${countdown}s 后重发` : '获取验证码'}
              </Button>
            </Space.Compact>
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[
              { required: true },
              { min: 8, message: '至少 8 位' },
            ]}
          >
            <Input.Password placeholder="至少 8 位" />
          </Form.Item>
          <Form.Item name="nickname" label="昵称">
            <Input placeholder="可选" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={submitting}>
            注册
          </Button>
        </Form>
        <div style={{ marginTop: 16, textAlign: 'center' }}>
          <Space>
            已有账号？<Link to="/login">立即登录</Link>
          </Space>
        </div>
      </Card>
    </div>
  );
}
