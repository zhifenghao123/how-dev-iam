import { useEffect, useState } from 'react';
import { Card, Button, Typography, message, Spin } from 'antd';
import { useParams, useNavigate } from 'react-router-dom';
import { apiAcceptInvitation } from '@/api';
import { useAuthStore } from '@/stores/authStore';

const { Title, Text } = Typography;

export default function InvitationAcceptPage() {
  const { token = '' } = useParams<{ token: string }>();
  const nav = useNavigate();
  const uid = useAuthStore((s) => s.userId);
  const [loading, setLoading] = useState(false);

  const accept = async () => {
    setLoading(true);
    try {
      const m = await apiAcceptInvitation(token);
      message.success('已加入企业');
      nav(`/accounts/${m.account_id}/members`);
    } catch {
      // toast handled
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!uid) {
      // 未登录，跳到登录并保留 token 待返回
      nav(`/login?next=/invitations/${token}`);
    }
  }, [uid, token, nav]);

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
      <Card style={{ width: 460, textAlign: 'center' }}>
        <Title level={3}>接受邀请</Title>
        <Text type="secondary">邀请令牌：{token.slice(0, 12)}...</Text>
        <div style={{ marginTop: 24 }}>
          <Button type="primary" onClick={accept} loading={loading} size="large">
            接受并加入
          </Button>
        </div>
      </Card>
    </div>
  );
}
