import { useState } from 'react';
import { Card, Form, Input, Button, Space, Typography, message } from 'antd';
import { useNavigate, Link } from 'react-router-dom';
import { apiRegisterPersonal } from '@/api';
import { useAuthStore } from '@/stores/authStore';

const { Title, Text } = Typography;

// 简化的"登录页"：iam-account 尚未提供登录接口（登录属 iam-authn 职责），
// 这里提供一个"我已经知道自己 user_id"的调试入口，用于本地打通闭环。
// 后续接入 iam-authn 后会替换为标准邮箱+密码登录。
export default function LoginPage() {
  const nav = useNavigate();
  const setUser = useAuthStore((s) => s.setUser);
  const [loading, setLoading] = useState(false);

  const onDebugLogin = async (v: { user_id: string; nickname: string }) => {
    const uid = Number(v.user_id);
    if (!uid) {
      message.error('user_id 必须是数字');
      return;
    }
    setUser(uid, v.nickname || 'user');
    message.success('已模拟登录');
    nav('/dashboard');
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
      <Card style={{ width: 420 }}>
        <Title level={3} style={{ textAlign: 'center' }}>
          how-dev-iam 登录
        </Title>
        <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
          正式登录接口由 iam-authn 提供；此处为最小闭环调试入口，注册成功后可直接进入。
        </Text>
        <Form layout="vertical" onFinish={onDebugLogin}>
          <Form.Item name="user_id" label="User ID（注册后自动获得）" rules={[{ required: true }]}>
            <Input placeholder="例如 12345678901234" />
          </Form.Item>
          <Form.Item name="nickname" label="昵称（本地显示用）">
            <Input placeholder="可选" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={loading}>
            进入
          </Button>
        </Form>
        <div style={{ marginTop: 16, textAlign: 'center' }}>
          <Space>
            没有账号？<Link to="/register">立即注册</Link>
          </Space>
        </div>
      </Card>
    </div>
  );
}
