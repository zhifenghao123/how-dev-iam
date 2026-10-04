import { useEffect, useState } from 'react';
import { List, Card, Tag, Button, Space, Typography, Empty, Spin } from 'antd';
import { useNavigate } from 'react-router-dom';
import { TeamOutlined, UserOutlined } from '@ant-design/icons';
import { apiListMyAccounts } from '@/api';
import type { MembershipView } from '@/api/types';
import { useAuthStore } from '@/stores/authStore';

const { Title, Text } = Typography;

export default function DashboardPage() {
  const nav = useNavigate();
  const setActive = useAuthStore((s) => s.setActiveAccount);
  const active = useAuthStore((s) => s.activeAccountId);
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<MembershipView[]>([]);

  const load = async () => {
    setLoading(true);
    try {
      const data = await apiListMyAccounts();
      setItems(data);
      if (!active && data.length > 0) {
        setActive(data[0].account_id);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 48 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <Card
      title={
        <Space>
          <TeamOutlined />
          <span>我的账户</span>
        </Space>
      }
    >
      {items.length === 0 ? (
        <Empty description="你还没有任何账户" />
      ) : (
        <List
          dataSource={items}
          renderItem={(m) => {
            const acc = m.account;
            const isPersonal = acc?.type === 'personal';
            return (
              <List.Item
                actions={[
                  m.role !== 'viewer' && !isPersonal && (
                    <Button
                      key="mgr"
                      type="link"
                      onClick={() => nav(`/accounts/${m.account_id}/members`)}
                    >
                      管理成员
                    </Button>
                  ),
                  <Button
                    key="switch"
                    type={active === m.account_id ? 'primary' : 'default'}
                    onClick={() => setActive(m.account_id)}
                  >
                    {active === m.account_id ? '当前工作空间' : '切换到此空间'}
                  </Button>,
                ].filter(Boolean) as any}
              >
                <List.Item.Meta
                  avatar={isPersonal ? <UserOutlined style={{ fontSize: 24 }} /> : <TeamOutlined style={{ fontSize: 24 }} />}
                  title={
                    <Space>
                      <span>{acc?.name || '(未知账户)'}</span>
                      <Tag color={isPersonal ? 'blue' : 'green'}>
                        {isPersonal ? '个人' : '企业'}
                      </Tag>
                      <Tag>{m.role}</Tag>
                      {acc?.verified && <Tag color="gold">已认证</Tag>}
                    </Space>
                  }
                  description={
                    <Space direction="vertical" size={2}>
                      <Text type="secondary">code: {acc?.code}</Text>
                      <Text type="secondary">
                        套餐 {acc?.plan} · 成员上限 {acc?.member_limit}
                      </Text>
                    </Space>
                  }
                />
              </List.Item>
            );
          }}
        />
      )}
    </Card>
  );
}
