import React, { useState } from 'react';
import {
  AuditOutlined,
  AppstoreOutlined,
  BellOutlined,
  BranchesOutlined,
  BugOutlined,
  LockOutlined,
  PlusCircleOutlined,
  SettingOutlined,
  TagsOutlined,
  TeamOutlined,
  ToolOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Badge, Button, Card, Dropdown, Form, Input, Layout, List, Menu, Modal, Space, Tag, Typography, message } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { changePassword } from '../api';
import { useAlerts } from '../context/AlertContext';
import { useAuth } from '../context/AuthContext';
import { severityColorMap, severityLabelMap } from '../constants/defects';

const { Sider, Header, Content } = Layout;
const { Title, Text } = Typography;

export default function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { criticalAlerts, criticalCount } = useAlerts();
  const { currentUser, hasRole, logout } = useAuth();
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [passwordSubmitting, setPasswordSubmitting] = useState(false);
  const [passwordForm] = Form.useForm();

  const menuItems = [
    { key: '/dashboard', icon: <AppstoreOutlined />, label: '数据看板' },
    { key: '/defects', icon: <BugOutlined />, label: '缺陷列表' },
    ...(hasRole('admin', 'inspector') ? [{ key: '/defects/new', icon: <PlusCircleOutlined />, label: '新增缺陷' }] : []),
    {
      key: 'master-data',
      icon: <SettingOutlined />,
      label: '基础数据管理',
      children: [
        { key: '/production-lines', icon: <BranchesOutlined />, label: '产线管理' },
        { key: '/equipment', icon: <ToolOutlined />, label: '设备管理' },
        { key: '/defect-types', icon: <TagsOutlined />, label: '缺陷类型管理' },
      ],
    },
    ...(hasRole('admin') ? [{ key: '/users', icon: <TeamOutlined />, label: '用户管理' }] : []),
    ...(hasRole('admin') ? [{ key: '/operation-logs', icon: <AuditOutlined />, label: '操作日志' }] : []),
  ];

  const flatMenuItems = menuItems.flatMap((item) => item.children || item);
  const selectedKey = flatMenuItems.find((item) =>
    location.pathname === item.key || location.pathname.startsWith(`${item.key}/`)
  )?.key || '/dashboard';

  const currentPageTitle = selectedKey === '/dashboard'
    ? '实时质量总览'
    : selectedKey === '/defects'
      ? '缺陷台账管理'
      : selectedKey === '/defects/new'
        ? '录入新缺陷'
        : selectedKey === '/production-lines'
          ? '产线基础数据管理'
          : selectedKey === '/equipment'
            ? '设备基础数据管理'
            : selectedKey === '/defect-types'
              ? '缺陷类型基础数据管理'
              : selectedKey === '/users'
                ? '系统用户管理'
                : selectedKey === '/operation-logs'
                  ? '操作日志审计'
                  : '缺陷详情追踪';

  const handleChangePassword = async (values) => {
    if (values.new_password !== values.confirm_password) {
      message.error('两次输入的新密码不一致');
      return;
    }
    setPasswordSubmitting(true);
    try {
      await changePassword({
        current_password: values.current_password,
        new_password: values.new_password,
      });
      message.success('密码修改成功');
      passwordForm.resetFields();
      setPasswordOpen(false);
    } catch (error) {
      message.error(error?.response?.data?.detail || '密码修改失败');
    } finally {
      setPasswordSubmitting(false);
    }
  };

  const alertOverlay = (
    <div
      style={{
        width: 360,
        background: '#0d1721',
        border: '1px solid rgba(255, 77, 79, 0.28)',
        borderRadius: 18,
        overflow: 'hidden',
        boxShadow: '0 18px 48px rgba(0, 0, 0, 0.35)',
      }}
    >
      <div style={{ padding: '14px 16px', borderBottom: '1px solid rgba(255,255,255,0.08)' }}>
        <Text style={{ color: '#ffd8d6', fontWeight: 700 }}>未处理严重缺陷</Text>
      </div>
      <List
        dataSource={criticalAlerts}
        locale={{ emptyText: <span style={{ color: '#9cb7ca' }}>暂无未处理严重缺陷</span> }}
        renderItem={(item) => (
          <List.Item
            style={{
              padding: '14px 16px',
              borderBlockEnd: '1px solid rgba(255,255,255,0.05)',
              cursor: 'pointer',
            }}
            onClick={() => navigate(`/defects/${item.id}`)}
          >
            <div style={{ width: '100%' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12 }}>
                <Text style={{ color: '#f8fbff', fontWeight: 600 }}>{item.name}</Text>
                <Tag color={severityColorMap[item.severity]}>{severityLabelMap[item.severity]}</Tag>
              </div>
              <Text style={{ color: '#aac5d8' }}>{item.defect_type}</Text>
              <div style={{ color: '#87a5bb', marginTop: 4 }}>{item.location || '位置未填写'}</div>
            </div>
          </List.Item>
        )}
      />
    </div>
  );

  return (
    <>
      <Layout style={{ minHeight: '100vh', background: '#081018' }}>
        <Sider width={252} theme="dark" style={{ background: 'linear-gradient(180deg, #07121d 0%, #0d1f2e 100%)', borderRight: '1px solid rgba(102, 217, 255, 0.16)' }}>
          <div style={{ padding: '28px 22px 20px', borderBottom: '1px solid rgba(102, 217, 255, 0.12)' }}>
            <Text style={{ color: '#66d9ff', fontSize: 12, letterSpacing: 3 }}>INDUSTRIAL QA</Text>
            <Title level={3} style={{ color: '#f3f7fb', margin: '10px 0 6px', fontFamily: 'Georgia, "Times New Roman", serif' }}>
              缺陷检测中心
            </Title>
            <Text style={{ color: 'rgba(219, 234, 254, 0.72)' }}>生产质检 · 可视化监控 · 全流程追踪</Text>
          </div>

          <div style={{ padding: '18px 18px 0' }}>
            <Menu
              mode="inline"
              theme="dark"
              selectedKeys={[selectedKey]}
              defaultOpenKeys={['master-data']}
              items={menuItems}
              onClick={({ key }) => navigate(key)}
              style={{ background: 'transparent', borderInlineEnd: 'none' }}
            />
          </div>
        </Sider>

        <Layout>
          <Header
            style={{
              minHeight: 88,
              height: 'auto',
              padding: '14px 28px',
              background: 'rgba(6, 17, 27, 0.92)',
              borderBottom: '1px solid rgba(102, 217, 255, 0.16)',
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              gap: 20,
              flexWrap: 'wrap',
            }}
          >
            <div style={{ minWidth: 260, flex: '1 1 420px' }}>
              <div style={{ color: '#8eb3c9', letterSpacing: 2, fontSize: 12, lineHeight: 1.6, whiteSpace: 'normal' }}>
                缺陷检测系统前端控制台
              </div>
              <div style={{ color: '#f8fbff', fontSize: 22, marginTop: 6, fontWeight: 600, lineHeight: 1.35, wordBreak: 'break-word' }}>
                {currentPageTitle}
              </div>
            </div>

            <div style={{ display: 'flex', alignItems: 'center', gap: 18, flexShrink: 0, flexWrap: 'wrap' }}>
              <Dropdown popupRender={() => alertOverlay} trigger={['click']} placement="bottomRight">
                <Badge count={criticalCount} offset={[-4, 4]}>
                  <Button
                    shape="circle"
                    icon={<BellOutlined />}
                    size="large"
                    style={{
                      background: criticalCount ? 'rgba(255, 77, 79, 0.16)' : 'rgba(255,255,255,0.06)',
                      borderColor: criticalCount ? 'rgba(255, 77, 79, 0.35)' : 'rgba(102, 217, 255, 0.14)',
                      color: criticalCount ? '#ff7875' : '#d6e8f5',
                    }}
                  />
                </Badge>
              </Dropdown>
              <Badge color="#52c41a" text={<span style={{ color: '#d6e8f5' }}>后端服务：localhost:8000</span>} />
              <Space size={10} style={{ background: 'rgba(255,255,255,0.04)', padding: '8px 12px', borderRadius: 999 }}>
                <UserOutlined style={{ color: '#66d9ff' }} />
                <Text style={{ color: '#f8fbff' }}>{currentUser?.display_name || currentUser?.username}</Text>
                <Tag color="cyan">{currentUser?.role_label || currentUser?.role}</Tag>
                <Button size="small" icon={<LockOutlined />} onClick={() => setPasswordOpen(true)}>修改密码</Button>
                <Button size="small" onClick={logout}>退出</Button>
              </Space>
            </div>
          </Header>

          <Content
            style={{
              padding: 28,
              background: 'radial-gradient(circle at top right, rgba(34, 89, 128, 0.24), transparent 26%), linear-gradient(180deg, #09131d 0%, #0e1722 100%)',
            }}
          >
            <Card style={{ marginBottom: 20, background: 'rgba(9, 28, 43, 0.92)', border: '1px solid rgba(102, 217, 255, 0.16)', borderRadius: 18 }}>
              <Space direction="vertical" size={2}>
                <Text style={{ color: '#f5fbff', fontWeight: 600 }}>演示登录信息</Text>
                <Text style={{ color: '#cfe0ec' }}>默认管理员账号：admin / Admin@123456</Text>
                <Text style={{ color: '#87a5bb' }}>登录后可在右上角“修改密码”，管理员可维护用户、日志与基础数据。</Text>
              </Space>
            </Card>
            <Outlet />
          </Content>
        </Layout>
      </Layout>

      <Modal
        open={passwordOpen}
        title="修改密码"
        onCancel={() => {
          setPasswordOpen(false);
          passwordForm.resetFields();
        }}
        onOk={() => passwordForm.submit()}
        confirmLoading={passwordSubmitting}
        destroyOnHidden
      >
        <Form form={passwordForm} layout="vertical" onFinish={handleChangePassword}>
          <Form.Item label="原密码" name="current_password" rules={[{ required: true, message: '请输入原密码' }]}>
            <Input.Password placeholder="请输入原密码" />
          </Form.Item>
          <Form.Item label="新密码" name="new_password" rules={[{ required: true, message: '请输入新密码' }]}>
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item label="确认新密码" name="confirm_password" rules={[{ required: true, message: '请再次输入新密码' }]}>
            <Input.Password placeholder="请再次输入新密码" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
