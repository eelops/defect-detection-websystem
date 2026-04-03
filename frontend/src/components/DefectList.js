import React, { useEffect, useState, useCallback } from 'react';
import {
  Layout, Table, Button, Tag, Space, Popconfirm, message, Select, Image,
} from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { fetchDefects, deleteDefect } from '../api';
import DefectForm from './DefectForm';

const { Header, Content } = Layout;

const severityColors = { low: 'blue', medium: 'orange', high: 'red', critical: 'magenta' };
const severityLabels = { low: '低', medium: '中', high: '高', critical: '严重' };
const statusColors = { open: 'default', fixing: 'processing', resolved: 'success', closed: 'default' };
const statusLabels = { open: '待处理', fixing: '修复中', resolved: '已解决', closed: '已关闭' };

export default function DefectList() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [editRecord, setEditRecord] = useState(null);
  const [filters, setFilters] = useState({});

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetchDefects(filters);
      setData(res.data);
    } catch {
      message.error('加载失败');
    }
    setLoading(false);
  }, [filters]);

  useEffect(() => { load(); }, [load]);

  const handleDelete = async (id) => {
    await deleteDefect(id);
    message.success('已删除');
    load();
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: '缺陷图片', dataIndex: 'image_path', width: 100,
      render: (v) => v ? <Image src={`http://localhost:8000${v}`} width={60} height={60} style={{ objectFit: 'cover' }} /> : '-',
    },
    { title: '名称', dataIndex: 'name', ellipsis: true },
    { title: '类型', dataIndex: 'defect_type', width: 90 },
    {
      title: '严重程度', dataIndex: 'severity', width: 90,
      render: (v) => <Tag color={severityColors[v]}>{severityLabels[v] || v}</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', width: 90,
      render: (v) => <Tag color={statusColors[v]}>{statusLabels[v] || v}</Tag>,
    },
    { title: '描述', dataIndex: 'description', ellipsis: true },
    {
      title: '创建时间', dataIndex: 'created_at', width: 170,
      render: (v) => new Date(v).toLocaleString('zh-CN'),
    },
    {
      title: '操作', width: 140, fixed: 'right',
      render: (_, r) => (
        <Space>
          <Button type="link" size="small" onClick={() => { setEditRecord(r); setFormOpen(true); }}>编辑</Button>
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <h1 style={{ color: '#fff', margin: 0, fontSize: 20 }}>缺陷检测管理系统</h1>
      </Header>
      <Content style={{ padding: 24 }}>
        <div style={{ marginBottom: 16, display: 'flex', gap: 12 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditRecord(null); setFormOpen(true); }}>
            新增缺陷
          </Button>
          <Select
            placeholder="按状态筛选" allowClear style={{ width: 140 }}
            onChange={(v) => setFilters((f) => ({ ...f, status: v }))}
            options={Object.entries(statusLabels).map(([k, v]) => ({ value: k, label: v }))}
          />
          <Select
            placeholder="按类型筛选" allowClear style={{ width: 140 }}
            onChange={(v) => setFilters((f) => ({ ...f, defect_type: v }))}
            options={['裂纹','划痕','变形','腐蚀','其他'].map((t) => ({ value: t, label: t }))}
          />
          <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
        </div>

        <Table
          rowKey="id"
          columns={columns}
          dataSource={data}
          loading={loading}
          scroll={{ x: 1000 }}
          pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
        />

        <DefectForm
          open={formOpen}
          record={editRecord}
          onClose={() => setFormOpen(false)}
          onSuccess={() => { setFormOpen(false); load(); }}
        />
      </Content>
    </Layout>
  );
}
