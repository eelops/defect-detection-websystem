import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Image,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import { ApiOutlined, DeleteOutlined, EditOutlined, EyeOutlined, ReloadOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { deleteDefect, fetchDefects, getImageUrl } from '../api';
import {
  defectTypes,
  severityColorMap,
  severityLabelMap,
  severityOptions,
  statusColorMap,
  statusLabelMap,
  statusOptions,
} from '../constants/defects';

const { Title, Text } = Typography;

const cardStyle = {
  background: 'linear-gradient(180deg, rgba(11, 26, 40, 0.96) 0%, rgba(16, 34, 51, 0.92) 100%)',
  border: '1px solid rgba(102, 217, 255, 0.14)',
  borderRadius: 20,
  boxShadow: '0 18px 40px rgba(0, 0, 0, 0.22)',
};

export default function DefectListPage() {
  const navigate = useNavigate();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [filters, setFilters] = useState({ defect_type: undefined, severity: undefined, status: undefined });

  const loadData = useCallback(async (page = pagination.current, pageSize = pagination.pageSize) => {
    setLoading(true);
    try {
      const result = await fetchDefects({
        ...filters,
        page,
        page_size: pageSize,
      });
      setItems(result.items || []);
      setPagination((prev) => ({ ...prev, current: page, pageSize, total: result.total || (result.items || []).length }));
    } catch (error) {
      message.error('缺陷列表加载失败');
    } finally {
      setLoading(false);
    }
  }, [filters, pagination.current, pagination.pageSize]);

  useEffect(() => {
    loadData(1, pagination.pageSize);
  }, [filters, loadData, pagination.pageSize]);

  const handleDelete = async (id) => {
    try {
      await deleteDefect(id);
      message.success('删除成功');
      loadData();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const columns = useMemo(() => ([
    { title: 'ID', dataIndex: 'id', width: 80 },
    {
      title: '图片',
      dataIndex: 'image_path',
      width: 110,
      render: (value) => value ? <Image width={64} height={64} src={getImageUrl(value)} style={{ objectFit: 'cover', borderRadius: 12 }} /> : <Text style={{ color: '#87a5bb' }}>暂无</Text>,
    },
    { title: '标题', dataIndex: 'name', ellipsis: true },
    { title: '类型', dataIndex: 'defect_type', width: 110 },
    {
      title: '等级',
      dataIndex: 'severity',
      width: 100,
      render: (value) => <Tag color={severityColorMap[value]}>{severityLabelMap[value] || value}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (value) => <Tag color={statusColorMap[value]}>{statusLabelMap[value] || value}</Tag>,
    },
    {
      title: '位置',
      dataIndex: 'location',
      width: 160,
      render: (value) => value || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
      render: (value) => value ? new Date(value).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'actions',
      width: 220,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button type="link" icon={<EyeOutlined />} onClick={() => navigate(`/defects/${record.id}`)}>查看详情</Button>
          <Button type="link" icon={<EditOutlined />} onClick={() => navigate(`/defects/${record.id}`)}>编辑</Button>
          <Popconfirm title="确认删除该缺陷吗？" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]), [navigate]);

  return (
    <Space direction="vertical" size={24} style={{ width: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', gap: 16, flexWrap: 'wrap' }}>
        <div>
          <Title level={2} style={{ color: '#f5fbff', margin: 0, fontFamily: 'Georgia, "Times New Roman", serif' }}>缺陷列表</Title>
          <Text style={{ color: 'rgba(216, 232, 246, 0.72)' }}>按类型、等级、状态筛选并维护缺陷台账。</Text>
        </div>
        <Button type="primary" onClick={() => navigate('/defects/new')}>新增缺陷</Button>
      </div>

      <Alert
        type="info"
        showIcon
        icon={<ApiOutlined />}
        message={<span style={{ color: '#f5fbff' }}>第三方缺陷 JSON 推送接口</span>}
        description={(
          <div style={{ color: '#dbe8f3', lineHeight: 1.8 }}>
            <div><strong>接口地址：</strong><code>POST http://localhost:8000/api/external/report</code></div>
            <div><strong>认证方式：</strong>Header 携带 <code>X-API-Key</code></div>
            <div><strong>数据格式：</strong>application/json，支持图片 base64、缺陷类型、等级、位置、置信度、来源等字段</div>
          </div>
        )}
        style={{ background: 'rgba(9, 28, 43, 0.92)', border: '1px solid rgba(102, 217, 255, 0.22)', borderRadius: 18 }}
      />

      <Card style={cardStyle} styles={{ body: { padding: 20 } }}>
        <Space wrap size={12} style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space wrap size={12}>
            <Select
              allowClear
              placeholder="按类型筛选"
              style={{ width: 160 }}
              options={defectTypes.map((item) => ({ value: item, label: item }))}
              onChange={(value) => setFilters((prev) => ({ ...prev, defect_type: value }))}
            />
            <Select
              allowClear
              placeholder="按等级筛选"
              style={{ width: 160 }}
              options={severityOptions}
              onChange={(value) => setFilters((prev) => ({ ...prev, severity: value }))}
            />
            <Select
              allowClear
              placeholder="按状态筛选"
              style={{ width: 160 }}
              options={statusOptions}
              onChange={(value) => setFilters((prev) => ({ ...prev, status: value }))}
            />
          </Space>

          <Button icon={<ReloadOutlined />} onClick={() => loadData(1, pagination.pageSize)}>刷新</Button>
        </Space>
      </Card>

      <Card style={cardStyle} styles={{ body: { padding: 8 } }}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={items}
          loading={loading}
          scroll={{ x: 1220 }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => loadData(page, pageSize),
          }}
        />
      </Card>
    </Space>
  );
}
