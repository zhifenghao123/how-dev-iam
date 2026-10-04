import { useState } from 'react';
import { Card, Form, Input, Button, Typography, message, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { apiRegisterOrganization } from '@/api';

const { Title } = Typography;

// 企业注册页（登录态）。
export default function RegisterOrganizationPage() {
  const nav = useNavigate();
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async (v: any) => {
    setSubmitting(true);
    try {
      const r = await apiRegisterOrganization(v);
      message.success('企业创建成功');
      nav(`/accounts/${r.account_id}/members`);
    } catch {
      // toast handled
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card title="创建企业账户" style={{ maxWidth: 720, margin: '0 auto' }}>
      <Form layout="vertical" onFinish={onSubmit}>
        <Form.Item name="name" label="企业显示名" rules={[{ required: true }]}>
          <Input placeholder="例如：Acme Inc." />
        </Form.Item>
        <Form.Item name="legal_name" label="企业注册名（可选）">
          <Input placeholder="工商注册全称" />
        </Form.Item>
        <Form.Item name="unified_credit_no" label="统一社会信用代码（可选）">
          <Input placeholder="18 位" />
        </Form.Item>
        <Form.Item name="industry" label="行业">
          <Input placeholder="互联网 / 金融 / ..." />
        </Form.Item>
        <Form.Item name="scale" label="规模">
          <Select
            allowClear
            options={[
              { value: '1-49', label: '1-49 人' },
              { value: '50-499', label: '50-499 人' },
              { value: '500+', label: '500+ 人' },
            ]}
          />
        </Form.Item>
        <Form.Item name="province" label="省份">
          <Input />
        </Form.Item>
        <Form.Item name="city" label="城市">
          <Input />
        </Form.Item>
        <Form.Item name="address" label="地址">
          <Input />
        </Form.Item>
        <Form.Item name="contact_email" label="联系邮箱">
          <Input />
        </Form.Item>
        <Form.Item name="contact_phone" label="联系电话">
          <Input />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={submitting}>
          提交
        </Button>
      </Form>
    </Card>
  );
}
