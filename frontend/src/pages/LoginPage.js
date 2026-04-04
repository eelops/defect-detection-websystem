import React, { useState } from 'react';
import { Button, Card, Form, Input, Typography, message } from 'antd';
import { useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

const { Title, Text } = Typography;

export default function LoginPage() {
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const location = useLocation();
  const { login } = useAuth();
  const [submitting, setSubmitting] = useState(false);

  const from = location.state?.from?.pathname || '/dashboard';

  const handleFinish = async (values) => {
    setSubmitting(true);
    try {
      await login(values);
      message.success('登录成功');
      navigate(from, { replace: true });
    } catch (error) {
      message.error(error?.response?.data?.detail || '登录失败');
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
        padding: 24,
        background: 'radial-gradient(circle at top, rgba(102,217,255,0.18), transparent 30%), linear-gradient(180deg, #081018 0%, #0c1722 100%)',
      }}
    >
      <Card
        style={{
          width: '100%',
          maxWidth: 460,
          background: 'linear-gradient(180deg, rgba(11, 26, 40, 0.96) 0%, rgba(16, 34, 51, 0.92) 100%)',
          border: '1px solid rgba(102, 217, 255, 0.14)',
          borderRadius: 20,
          boxShadow: '0 18px 40px rgba(0, 0, 0, 0.22)',
        }}
        styles={{ body: { padding: 28 } }}
      >
        <div style={{ marginBottom: 24 }}>
          <Text style={{ color: '#66d9ff', letterSpacing: 3, fontSize: 12 }}>INDUSTRIAL QA</Text>
          <Title level={2} style={{ color: '#f5fbff', margin: '10px 0 6px', fontFamily: 'Georgia, "Times New Roman", serif' }}>
            缺陷检测中心登录
          </Title>
          <Text style={{ color: 'rgba(216, 232, 246, 0.72)' }}>
            请输入账号密码进入系统。默认管理员账号为 admin / Admin@123456。
          </Text>
        </div>

        <Form form={form} layout="vertical" onFinish={handleFinish}>
          <Form.Item label={<span style={{ color: '#dbe8f3' }}>用户名</span>} name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input size="large" placeholder="请输入用户名" />
          </Form.Item>
          <Form.Item label={<span style={{ color: '#dbe8f3' }}>密码</span>} name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password size="large" placeholder="请输入密码" />
          </Form.Item>
          <Button type="primary" htmlType="submit" size="large" loading={submitting} block>
            登录系统
          </Button>
        </Form>
      </Card>
    </div>
  );
}
