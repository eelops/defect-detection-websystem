import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Descriptions,
  Image,
  Select,
  Skeleton,
  Space,
  Tag,
  Typography,
  message,
} from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { fetchDefect, getImageUrl, updateDefect } from '../api';
import {
  severityColorMap,
  severityLabelMap,
  statusColorMap,
  statusLabelMap,
  statusOptions,
} from '../constants/defects';

const { Title, Text, Paragraph } = Typography;

const cardStyle = {
  background: 'linear-gradient(180deg, rgba(11, 26, 40, 0.96) 0%, rgba(16, 34, 51, 0.92) 100%)',
  border: '1px solid rgba(102, 217, 255, 0.14)',
  borderRadius: 20,
  boxShadow: '0 18px 40px rgba(0, 0, 0, 0.22)',
};

export default function DefectDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(false);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const result = await fetchDefect(id);
        setDetail(result);
      } catch (error) {
        message.error('缺陷详情加载失败');
      } finally {
        setLoading(false);
      }
    };

    load();
  }, [id]);

  const handleStatusChange = async (status) => {
    setUpdating(true);
    try {
      await updateDefect(id, { status });
      setDetail((prev) => ({ ...prev, status }));
      message.success('状态更新成功');
    } catch (error) {
      message.error('状态更新失败');
    } finally {
      setUpdating(false);
    }
  };

  if (loading) {
    return <Skeleton active paragraph={{ rows: 8 }} />;
  }

  if (!detail) {
    return (
      <Card style={cardStyle}>
        <Text style={{ color: '#e8f3fb' }}>未找到对应缺陷。</Text>
      </Card>
    );
  }

  return (
    <Space direction="vertical" size={24} style={{ width: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 16, flexWrap: 'wrap' }}>
        <div>
          <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => navigate('/defects')} style={{ paddingLeft: 0 }}>
            返回列表
          </Button>
          <Title level={2} style={{ color: '#f5fbff', margin: 0, fontFamily: 'Georgia, "Times New Roman", serif' }}>{detail.name || `缺陷 #${id}`}</Title>
          <Space size={12} style={{ marginTop: 10 }}>
            <Tag color={severityColorMap[detail.severity]}>{severityLabelMap[detail.severity] || detail.severity}</Tag>
            <Tag color={statusColorMap[detail.status]}>{statusLabelMap[detail.status] || detail.status}</Tag>
          </Space>
        </div>

        <Space>
          <Text style={{ color: '#dcebf7' }}>处理状态</Text>
          <Select
            value={detail.status}
            options={statusOptions}
            loading={updating}
            style={{ width: 160 }}
            onChange={handleStatusChange}
          />
        </Space>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(320px, 1.4fr) minmax(280px, 1fr)', gap: 20 }}>
        <Card title={<span style={{ color: '#e8f3fb' }}>缺陷信息</span>} style={cardStyle} styles={{ header: { borderBottomColor: 'rgba(102, 217, 255, 0.12)' } }}>
          <Descriptions column={1} labelStyle={{ color: '#89a8bf', width: 120 }} contentStyle={{ color: '#f5fbff' }}>
            <Descriptions.Item label="标题">{detail.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="类型">{detail.defect_type || '-'}</Descriptions.Item>
            <Descriptions.Item label="等级">{severityLabelMap[detail.severity] || detail.severity || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">{statusLabelMap[detail.status] || detail.status || '-'}</Descriptions.Item>
            <Descriptions.Item label="位置">{detail.location || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{detail.created_at ? new Date(detail.created_at).toLocaleString('zh-CN') : '-'}</Descriptions.Item>
            <Descriptions.Item label="描述">
              <Paragraph style={{ marginBottom: 0, color: '#f5fbff' }}>{detail.description || '暂无描述'}</Paragraph>
            </Descriptions.Item>
          </Descriptions>
        </Card>

        <Card title={<span style={{ color: '#e8f3fb' }}>缺陷图片</span>} style={cardStyle} styles={{ header: { borderBottomColor: 'rgba(102, 217, 255, 0.12)' } }}>
          {detail.image_path ? (
            <Image src={getImageUrl(detail.image_path)} style={{ width: '100%', borderRadius: 16 }} />
          ) : (
            <Text style={{ color: '#9cb7ca' }}>暂无图片</Text>
          )}
        </Card>
      </div>
    </Space>
  );
}
