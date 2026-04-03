import React from 'react';
import {
  AppstoreOutlined,
  BellOutlined,
  BugOutlined,
  PlusCircleOutlined,
} from '@ant-design/icons';
import { Badge, Button, Dropdown, Layout, List, Menu, Tag, Typography } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAlerts } from '../context/AlertContext';
import { severityColorMap, severityLabelMap } from '../constants/defects';

const { Sider, Header, Content } = Layout;
const { Title, Text } = Typography;

const menuItems = [
  { key: '/dashboard', icon: <AppstoreOutlined />, label: '数据看板' },
  { key: '/defects', icon: <BugOutlined />, label: '缺陷列表' },
  { key: '/defects/new', icon: <PlusCircleOutlined />, label: '新增缺陷' },
];

export default function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { criticalAlerts, criticalCount } = useAlerts();

  const selectedKey = menuItems.find((item) =>
    location.pathname === item.key || location.pathname.startsWith(`${item.key}/`)
  )?.key || '/dashboard';

  const currentPageTitle = selectedKey === '/dashboard'
    ? '实时质量总览'
    : selectedKey === '/defects'
      ? '缺陷台账管理'
      : selectedKey === '/defects/new'
        ? '录入新缺陷'
        : '缺陷详情追踪';

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
          </div>
        </Header>

        <Content
          style={{
            padding: 28,
            background: 'radial-gradient(circle at top right, rgba(34, 89, 128, 0.24), transparent 26%), linear-gradient(180deg, #09131d 0%, #0e1722 100%)',
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
