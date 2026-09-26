import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Card, Col, Empty, Row, Space, Statistic, Typography, message } from 'antd';
import { AlertOutlined, BugOutlined, FireOutlined, SafetyCertificateOutlined, DashboardOutlined } from '@ant-design/icons';
import { Line, Pie } from '@ant-design/charts';
import dayjs from 'dayjs';
import { fetchDefects } from '../api';
import { severityLabelMap, severityOptions, statusLabelMap } from '../constants/defects';

const { Title, Text } = Typography;

const chartPalette = ['#2563eb', '#16a34a', '#d97706', '#dc2626', '#7c3aed', '#0891b2'];

function buildTrend(items) {
  const map = new Map();
  for (let i = 6; i >= 0; i -= 1) {
    const date = dayjs().subtract(i, 'day').format('YYYY-MM-DD');
    map.set(date, 0);
  }

  items.forEach((item) => {
    const key = item.created_at ? dayjs(item.created_at).format('YYYY-MM-DD') : null;
    if (key && map.has(key)) {
      map.set(key, map.get(key) + 1);
    }
  });

  return Array.from(map.entries()).map(([date, value]) => ({ date, value }));
}

export default function DashboardPage() {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const result = await fetchDefects();
        setItems(result.items || []);
      } catch (error) {
        message.error('看板数据加载失败');
      } finally {
        setLoading(false);
      }
    };

    load();
  }, []);

  const severityStats = useMemo(() => {
    const base = Object.fromEntries(severityOptions.map((item) => [item.value, 0]));
    items.forEach((item) => {
      if (base[item.severity] !== undefined) {
        base[item.severity] += 1;
      }
    });
    return base;
  }, [items]);

  const criticalOpenItems = useMemo(
    () => items.filter((item) => item.severity === 'critical' && ['open', 'assigned', 'processing', 'pending_acceptance', 'rejected'].includes(item.status)),
    [items]
  );

  const pieData = useMemo(() => {
    const counter = new Map();
    items.forEach((item) => {
      const key = item.defect_type || '未知';
      counter.set(key, (counter.get(key) || 0) + 1);
    });
    return Array.from(counter.entries()).map(([type, value]) => ({ type, value }));
  }, [items]);

  const lineData = useMemo(() => buildTrend(items), [items]);

  return (
    <Space direction="vertical" size={20} className="page-stack">
      <div className="page-header">
        <div className="page-heading">
          <Title level={2} className="page-heading-title">数据看板</Title>
          <Text className="page-heading-subtitle">汇总缺陷数量、风险等级分布与近 7 天趋势。</Text>
        </div>
      </div>

      {criticalOpenItems.length > 0 && (
        <Alert
          type="error"
          showIcon
          icon={<AlertOutlined />}
          message={<span style={{ fontWeight: 700 }}>当前存在 {criticalOpenItems.length} 条未关闭严重缺陷</span>}
          description={(
            <Space direction="vertical" size={6} style={{ width: '100%' }}>
              {criticalOpenItems.map((item) => (
                <Text key={item.id}>
                  #{item.id} {item.name} ｜ {item.location || '位置未填写'} ｜ {statusLabelMap[item.status] || item.status}
                </Text>
              ))}
            </Space>
          )}
        />
      )}

      <Row gutter={[18, 18]}>
        <Col xs={24} md={12} xl={6}>
          <Card loading={loading} className="stat-card">
            <div className="metric-card">
              <Statistic title="缺陷总数" value={items.length} />
              <div className="metric-icon"><DashboardOutlined /></div>
            </div>
          </Card>
        </Col>
        <Col xs={24} md={12} xl={6}>
          <Card loading={loading} className="stat-card">
            <div className="metric-card">
              <Statistic title="严重缺陷" value={severityStats.critical} />
              <div className="metric-icon danger"><FireOutlined /></div>
            </div>
          </Card>
        </Col>
        <Col xs={24} md={12} xl={6}>
          <Card loading={loading} className="stat-card">
            <div className="metric-card">
              <Statistic title="高等级缺陷" value={severityStats.high} />
              <div className="metric-icon warning"><BugOutlined /></div>
            </div>
          </Card>
        </Col>
        <Col xs={24} md={12} xl={6}>
          <Card loading={loading} className="stat-card">
            <div className="metric-card">
              <Statistic title="已解决缺陷" value={items.filter((item) => item.status === 'closed').length} />
              <div className="metric-icon success"><SafetyCertificateOutlined /></div>
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[18, 18]}>
        <Col xs={24} xl={10}>
          <Card title="各等级数量" loading={loading} className="content-card">
            <Space direction="vertical" style={{ width: '100%' }} size={14}>
              {severityOptions.map((item) => (
                <div key={item.value} className="severity-row">
                  <Text>{severityLabelMap[item.value]}</Text>
                  <Text style={{ fontSize: 18, fontWeight: 700 }}>{severityStats[item.value]}</Text>
                </div>
              ))}
            </Space>
          </Card>
        </Col>

        <Col xs={24} xl={14}>
          <Card title="缺陷类型分布" loading={loading} className="content-card">
            {pieData.length ? (
              <Pie
                data={pieData}
                angleField="value"
                colorField="type"
                innerRadius={0.55}
                legend={{ position: 'bottom', itemName: { style: { fill: '#475467' } } }}
                label={{ text: 'value', style: { fill: '#1f2937', fontWeight: 600 } }}
                tooltip={{ items: [{ channel: 'y', name: '数量' }] }}
                scale={{ color: { range: chartPalette } }}
              />
            ) : (
              <Empty description="暂无图表数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>
        </Col>
      </Row>

      <Card title="近 7 天缺陷趋势" loading={loading} className="content-card">
        {lineData.length ? (
          <Line
            data={lineData}
            xField="date"
            yField="value"
            point={{ size: 5, shape: 'diamond', style: { fill: '#2563eb', stroke: '#ffffff', lineWidth: 2 } }}
            style={{ lineWidth: 3, stroke: '#2563eb' }}
            axis={{ x: { labelFill: '#667085' }, y: { labelFill: '#667085' } }}
            tooltip={{ items: [{ channel: 'y', name: '新增缺陷' }] }}
            smooth
          />
        ) : (
          <Empty description="暂无趋势数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
      </Card>
    </Space>
  );
}
