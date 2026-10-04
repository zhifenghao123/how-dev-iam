import { useEffect, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  message,
  Tag,
  Typography,
  Divider,
  Alert,
} from 'antd';
import { useParams } from 'react-router-dom';
import {
  apiListMembers,
  apiCreateInvitation,
  apiCreateMember,
  apiRemoveMember,
} from '@/api';
import type { MembershipView, InvitationView } from '@/api/types';

const { Text, Paragraph } = Typography;

export default function MembersPage() {
  const { aid } = useParams<{ aid: string }>();
  const accountId = Number(aid);

  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<MembershipView[]>([]);

  // 邀请弹窗
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteForm] = Form.useForm();
  const [inviteResult, setInviteResult] = useState<InvitationView | null>(null);

  // 直接创建子用户弹窗
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const data = await apiListMembers(accountId);
      setRows(data);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (accountId) load();
  }, [accountId]);

  const onInvite = async (v: any) => {
    try {
      const inv = await apiCreateInvitation(accountId, {
        email: v.email,
        role: v.role,
      });
      setInviteResult(inv);
      message.success('邀请已生成');
    } catch {
      // toast handled
    }
  };

  const onCreate = async (v: any) => {
    try {
      await apiCreateMember(accountId, v);
      message.success('已创建子用户');
      setCreateOpen(false);
      createForm.resetFields();
      load();
    } catch {
      // toast handled
    }
  };

  const onRemove = (m: MembershipView) => {
    Modal.confirm({
      title: '确认移除该成员？',
      content: `将把 ${m.display_name || m.user?.email || m.user_id} 从当前企业移除。`,
      okType: 'danger',
      onOk: async () => {
        await apiRemoveMember(accountId, m.id);
        message.success('已移除');
        load();
      },
    });
  };

  const inviteLink = inviteResult
    ? `${window.location.origin}/invitations/${inviteResult.token}`
    : '';

  const columns = [
    {
      title: '成员',
      key: 'user',
      render: (_: any, r: MembershipView) =>
        r.user ? (
          <Space direction="vertical" size={0}>
            <span>{r.display_name || r.user.nickname || r.user.email || r.user.id}</span>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {r.user.email || r.user.phone || `uid=${r.user.id}`}
            </Text>
          </Space>
        ) : (
          <span>uid={r.user_id}</span>
        ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      render: (v: string) => <Tag color={v === 'owner' ? 'gold' : v === 'admin' ? 'blue' : 'default'}>{v}</Tag>,
    },
    { title: '来源', dataIndex: 'join_source' },
    {
      title: '状态',
      dataIndex: 'status',
      render: (v: string) => <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag>,
    },
    { title: '加入时间', dataIndex: 'join_time' },
    {
      title: '操作',
      key: 'op',
      render: (_: any, r: MembershipView) =>
        r.role === 'owner' ? (
          <Text type="secondary">Owner 不可移除</Text>
        ) : (
          <Button danger type="link" onClick={() => onRemove(r)}>
            移除
          </Button>
        ),
    },
  ];

  return (
    <Card
      title={`企业成员管理（Account #${accountId}）`}
      extra={
        <Space>
          <Button onClick={() => setInviteOpen(true)}>发送邀请链接</Button>
          <Button type="primary" onClick={() => setCreateOpen(true)}>
            直接创建子用户
          </Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        loading={loading}
        columns={columns as any}
        dataSource={rows}
        pagination={false}
      />

      {/* 邀请弹窗 */}
      <Modal
        title="邀请成员"
        open={inviteOpen}
        onCancel={() => {
          setInviteOpen(false);
          setInviteResult(null);
          inviteForm.resetFields();
        }}
        footer={null}
      >
        <Form form={inviteForm} layout="vertical" onFinish={onInvite}>
          <Form.Item
            name="email"
            label="邮箱"
            rules={[{ required: true, type: 'email' }]}
          >
            <Input placeholder="被邀请人邮箱" />
          </Form.Item>
          <Form.Item name="role" label="角色" initialValue="member">
            <Select
              options={[
                { value: 'admin', label: 'admin' },
                { value: 'member', label: 'member' },
                { value: 'viewer', label: 'viewer' },
              ]}
            />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            生成邀请链接
          </Button>
        </Form>
        {inviteResult && (
          <>
            <Divider />
            <Alert
              type="success"
              message="邀请链接已生成"
              description={
                <Paragraph copyable={{ text: inviteLink }} style={{ marginBottom: 0 }}>
                  {inviteLink}
                </Paragraph>
              }
            />
          </>
        )}
      </Modal>

      {/* 创建子用户弹窗 */}
      <Modal
        title="直接创建子用户"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
      >
        <Form form={createForm} layout="vertical" onFinish={onCreate}>
          <Form.Item
            name="email"
            label="邮箱"
            rules={[{ required: true, type: 'email' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="password"
            label="初始密码"
            rules={[{ required: true, min: 8 }]}
            extra="首次登录时强制修改"
          >
            <Input.Password />
          </Form.Item>
          <Form.Item name="real_name" label="真实姓名">
            <Input />
          </Form.Item>
          <Form.Item name="nickname" label="昵称">
            <Input />
          </Form.Item>
          <Form.Item name="role" label="角色" initialValue="member">
            <Select
              options={[
                { value: 'admin', label: 'admin' },
                { value: 'member', label: 'member' },
                { value: 'viewer', label: 'viewer' },
              ]}
            />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            创建
          </Button>
        </Form>
      </Modal>
    </Card>
  );
}
