import React, { useState } from 'react';
import { Button, Card, Form, Input, Typography, message } from 'antd';
import { SafetyCertificateOutlined } from '@ant-design/icons';
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
    <div className="login-shell">
      <div className="login-panel">
        <Card className="login-card" styles={{ body: { padding: 0 } }}>
          <div className="login-brand-row">
            <div className="login-brand-mark">
              <SafetyCertificateOutlined />
            </div>
            <div>
              <Text type="secondary">Industrial QA</Text>
              <Title level={2} style={{ margin: '2px 0 0' }}>缺陷检测中心</Title>
            </div>
          </div>

          <div style={{ marginBottom: 24 }}>
            <Text type="secondary">默认管理员账号：admin / Admin@123456</Text>
          </div>

          <Form form={form} layout="vertical" onFinish={handleFinish}>
            <Form.Item label="用户名" name="username" rules={[{ required: true, message: '请输入用户名' }]}>
              <Input size="large" placeholder="请输入用户名" />
            </Form.Item>
            <Form.Item label="密码" name="password" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password size="large" placeholder="请输入密码" />
            </Form.Item>
            <Button type="primary" htmlType="submit" size="large" loading={submitting} block>
              登录系统
            </Button>
          </Form>
        </Card>
      </div>

      <div className="login-side">
        <div className="login-status-board">
          <Title level={1} className="login-board-title">工业缺陷管理系统</Title>
          <div className="login-board-grid">
            <div className="login-board-item">
              <Text style={{ color: 'rgba(255,255,255,0.72)' }}>流程状态</Text>
              <strong>闭环追踪</strong>
            </div>
            <div className="login-board-item">
              <Text style={{ color: 'rgba(255,255,255,0.72)' }}>告警级别</Text>
              <strong>严重优先</strong>
            </div>
            <div className="login-board-item">
              <Text style={{ color: 'rgba(255,255,255,0.72)' }}>审计范围</Text>
              <strong>全量留痕</strong>
            </div>
            <div className="login-board-item">
              <Text style={{ color: 'rgba(255,255,255,0.72)' }}>数据出口</Text>
              <strong>Excel / PDF</strong>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
