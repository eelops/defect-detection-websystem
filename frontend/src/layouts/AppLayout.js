import React, { useState } from 'react';
import {
  AuditOutlined,
  AppstoreOutlined,
  BellOutlined,
  BranchesOutlined,
  BugOutlined,
  LogoutOutlined,
  LockOutlined,
  PlusCircleOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  TagsOutlined,
  TeamOutlined,
  ToolOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Avatar, Badge, Button, Dropdown, Form, Input, Layout, List, Menu, Modal, Space, Tag, Typography, message } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { changePassword } from '../api';
import { useAlerts } from '../context/AlertContext';
import { useAuth } from '../context/AuthContext';
import { severityColorMap, severityLabelMap } from '../constants/defects';

const { Sider, Header, Content } = Layout;
const { Text } = Typography;

export default function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { criticalAlerts, criticalCount, alertError } = useAlerts();
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
    <div className="alert-dropdown">
      <div className="alert-dropdown-header">
        <Text>未处理严重缺陷</Text>
      </div>
      <List
        dataSource={criticalAlerts}
        locale={{ emptyText: <span style={{ color: alertError ? '#dc2626' : '#667085' }}>{alertError || '暂无未处理严重缺陷'}</span> }}
        renderItem={(item) => (
          <List.Item
            style={{
              padding: '14px 16px',
              borderBlockEnd: '1px solid #e8edf3',
              cursor: 'pointer',
            }}
            onClick={() => navigate(`/defects/${item.id}`)}
          >
            <div style={{ width: '100%' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12 }}>
                <Text strong>{item.name}</Text>
                <Tag color={severityColorMap[item.severity]}>{severityLabelMap[item.severity]}</Tag>
              </div>
              <Text type="secondary">{item.defect_type}</Text>
              <div style={{ color: '#667085', marginTop: 4 }}>{item.location || '位置未填写'}</div>
            </div>
          </List.Item>
        )}
      />
    </div>
  );

  return (
    <>
      <Layout className="app-shell">
        <Sider width={252} theme="dark" className="app-sider">
          <div className="brand-block">
            <div className="brand-row">
              <div className="brand-mark">
                <SafetyCertificateOutlined />
              </div>
              <div>
                <Text className="brand-kicker">Industrial QA</Text>
                <Text className="brand-title">缺陷检测中心</Text>
              </div>
            </div>
            <Text className="brand-subtitle">生产质检 / 流程闭环 / 安全审计</Text>
          </div>

          <div className="side-menu-wrap">
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
          <Header className="app-header">
            <div className="page-title-block">
              <div className="page-kicker">缺陷检测系统前端控制台</div>
              <div className="page-current-title">
                {currentPageTitle}
              </div>
            </div>

            <div className="header-actions">
              <Dropdown popupRender={() => alertOverlay} trigger={['click']} placement="bottomRight">
                <Badge count={criticalCount} offset={[-4, 4]}>
                  <Button
                    shape="circle"
                    icon={<BellOutlined />}
                    className="icon-button"
                    danger={criticalCount > 0}
                  />
                </Badge>
              </Dropdown>
              <Badge className="service-badge" color="#16a34a" text={<span>后端服务：localhost:8000</span>} />
              <Space size={8} className="user-chip">
                <Avatar size={28} style={{ background: '#eef4ff', color: '#2563eb' }} icon={<UserOutlined />} />
                <Text strong>{currentUser?.display_name || currentUser?.username}</Text>
                <Tag color="cyan">{currentUser?.role_label || currentUser?.role}</Tag>
                <Button size="small" icon={<LockOutlined />} onClick={() => setPasswordOpen(true)}>修改密码</Button>
                <Button size="small" icon={<LogoutOutlined />} onClick={logout}>退出</Button>
              </Space>
            </div>
          </Header>

          <Content className="page-content">
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
