import React, { useEffect, useMemo, useState } from 'react';
import { Card, Col, Empty, Row, Space, Statistic, Typography, message } from 'antd';
import { BugOutlined, FireOutlined, SafetyCertificateOutlined, DashboardOutlined } from '@ant-design/icons';
import { Line, Pie } from '@ant-design/charts';
import dayjs from 'dayjs';
import { fetchDefects } from '../api';
import { severityLabelMap, severityOptions } from '../constants/defects';

const { Title, Text } = Typography;

const cardStyle = {
  background: 'linear-gradient(180deg, rgba(11, 26, 40, 0.96) 0%, rgba(16, 34, 51, 0.92) 100%)',
  border: '1px solid rgba(102, 217, 255, 0.14)',
  borderRadius: 20,
  boxShadow: '0 18px 40px rgba(0, 0, 0, 0.22)',
};

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
    <Space direction="vertical" size={24} style={{ width: '100%' }}>
      <div>
        <Title level={2} style={{ color: '#f5fbff', margin: 0, fontFamily: 'Georgia, "Times New Roman", serif' }}>
          数据看板
        </Title>
        <Text style={{ color: 'rgba(216, 232, 246, 0.72)' }}>汇总缺陷数量、风险等级分布与近 7 天趋势。</Text>
      </div>

      <Row gutter={[18, 18]}>
        <Col xs={24} md={12} xl={6}>
          <Card loading={loading} style={cardStyle} styles={{ body: { padding: 22 } }}>
            <Statistic title={<span style={{ color: '#89a8bf' }}>缺陷总数</span>} value={items.length} prefix={<DashboardOutlined style={{ color: '#66d9ff' }} />} valueStyle={{ color: '#f3fbff' }} />
          </Card>
        </Col>
        <Col xs={24} md={12} xl={6}>
          <Card loading={loading} style={cardStyle} styles={{ body: { padding: 22 } }}>
            <Statistic title={<span style={{ color: '#89a8bf' }}>严重缺陷</span>} value={severityStats.critical} prefix={<FireOutlined style={{ color: '#ff4d4f' }} />} valueStyle={{ color: '#fff1f0' }} />
          </Card>
        </Col>
        <Col xs={24} md={12} xl={6}>
          <Card loading={loading} style={cardStyle} styles={{ body: { padding: 22 } }}>
            <Statistic title={<span style={{ color: '#89a8bf' }}>高等级缺陷</span>} value={severityStats.high} prefix={<BugOutlined style={{ color: '#fa8c16' }} />} valueStyle={{ color: '#fff7e6' }} />
          </Card>
        </Col>
        <Col xs={24} md={12} xl={6}>
          <Card loading={loading} style={cardStyle} styles={{ body: { padding: 22 } }}>
            <Statistic title={<span style={{ color: '#89a8bf' }}>已解决缺陷</span>} value={items.filter((item) => item.status === 'resolved').length} prefix={<SafetyCertificateOutlined style={{ color: '#52c41a' }} />} valueStyle={{ color: '#f6ffed' }} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[18, 18]}>
        <Col xs={24} xl={10}>
          <Card title={<span style={{ color: '#e8f3fb' }}>各等级数量</span>} loading={loading} style={cardStyle} styles={{ header: { borderBottomColor: 'rgba(102, 217, 255, 0.12)' } }}>
            <Space direction="vertical" style={{ width: '100%' }} size={14}>
              {severityOptions.map((item) => (
                <div key={item.value} style={{ display: 'flex', justifyContent: 'space-between', padding: '12px 14px', borderRadius: 14, background: 'rgba(255,255,255,0.03)' }}>
                  <Text style={{ color: '#dbe8f3' }}>{severityLabelMap[item.value]}</Text>
                  <Text style={{ color: '#ffffff', fontSize: 18, fontWeight: 700 }}>{severityStats[item.value]}</Text>
                </div>
              ))}
            </Space>
          </Card>
        </Col>

        <Col xs={24} xl={14}>
          <Card title={<span style={{ color: '#e8f3fb' }}>缺陷类型分布</span>} loading={loading} style={cardStyle} styles={{ header: { borderBottomColor: 'rgba(102, 217, 255, 0.12)' } }}>
            {pieData.length ? (
              <Pie
                data={pieData}
                angleField="value"
                colorField="type"
                innerRadius={0.55}
                legend={{ position: 'bottom', itemName: { style: { fill: '#dbe8f3' } } }}
                label={{ text: 'value', style: { fill: '#fff', fontWeight: 600 } }}
                tooltip={{ items: [{ channel: 'y', name: '数量' }] }}
                scale={{ color: { range: ['#66d9ff', '#52c41a', '#faad14', '#ff7875', '#9254de', '#13c2c2'] } }}
              />
            ) : (
              <Empty description="暂无图表数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>
        </Col>
      </Row>

      <Card title={<span style={{ color: '#e8f3fb' }}>近 7 天缺陷趋势</span>} loading={loading} style={cardStyle} styles={{ header: { borderBottomColor: 'rgba(102, 217, 255, 0.12)' } }}>
        {lineData.length ? (
          <Line
            data={lineData}
            xField="date"
            yField="value"
            point={{ size: 5, shape: 'diamond', style: { fill: '#66d9ff', stroke: '#09131d', lineWidth: 2 } }}
            style={{ lineWidth: 3, stroke: '#66d9ff' }}
            axis={{ x: { labelFill: '#aac6d9' }, y: { labelFill: '#aac6d9' } }}
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
