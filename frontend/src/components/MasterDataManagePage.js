import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Form, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;

const cardStyle = {
  background: 'linear-gradient(180deg, rgba(11, 26, 40, 0.96) 0%, rgba(16, 34, 51, 0.92) 100%)',
  border: '1px solid rgba(102, 217, 255, 0.14)',
  borderRadius: 20,
  boxShadow: '0 18px 40px rgba(0, 0, 0, 0.22)',
};

const statusOptions = [
  { value: 'active', label: '启用' },
  { value: 'inactive', label: '停用' },
];

export default function MasterDataManagePage({
  title,
  subtitle,
  fields,
  fetchList,
  createItem,
  updateItem,
  buildPayload,
}) {
  const [form] = Form.useForm();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState(undefined);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const result = await fetchList({ keyword: keyword || undefined, status });
      setItems(result.items || []);
    } catch (error) {
      message.error('基础数据加载失败');
    } finally {
      setLoading(false);
    }
  }, [fetchList, keyword, status]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const columns = useMemo(() => ([
    { title: '编码', dataIndex: 'code', width: 140 },
    { title: '名称', dataIndex: 'name', width: 180 },
    ...fields.filter((field) => field.table).map((field) => ({
      title: field.label,
      dataIndex: field.name,
      render: field.render,
    })),
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (value) => <Tag color={value === 'active' ? 'success' : 'default'}>{value === 'active' ? '启用' : '停用'}</Tag>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 140,
      render: (_, record) => (
        <Button type="link" onClick={() => {
          setEditing(record);
          form.setFieldsValue({ ...record, status: record.status || 'active' });
          setOpen(true);
        }}>
          编辑
        </Button>
      ),
    },
  ]), [fields, form]);

  const handleSubmit = async (values) => {
    setSaving(true);
    try {
      const payload = buildPayload ? buildPayload(values) : values;
      if (editing) {
        await updateItem(editing.id, payload);
        message.success('更新成功');
      } else {
        await createItem(payload);
        message.success('新增成功');
      }
      setOpen(false);
      setEditing(null);
      form.resetFields();
      loadData();
    } catch (error) {
      message.error(error?.response?.data?.detail || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', gap: 16, flexWrap: 'wrap' }}>
        <div>
          <Title level={2} style={{ color: '#f5fbff', margin: 0, fontFamily: 'Georgia, "Times New Roman", serif' }}>{title}</Title>
          <Text style={{ color: 'rgba(216, 232, 246, 0.72)' }}>{subtitle}</Text>
        </div>
        <Button type="primary" onClick={() => { setEditing(null); form.setFieldsValue({ status: 'active' }); setOpen(true); }}>新增</Button>
      </div>

      <Card style={cardStyle} styles={{ body: { padding: 20 } }}>
        <Space wrap size={12} style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space wrap size={12}>
            <Input placeholder="按编码/名称搜索" style={{ width: 240 }} value={keyword} onChange={(e) => setKeyword(e.target.value)} />
            <Select allowClear placeholder="按状态筛选" style={{ width: 160 }} options={statusOptions} value={status} onChange={setStatus} />
          </Space>
          <Button icon={<ReloadOutlined />} onClick={loadData}>刷新</Button>
        </Space>
      </Card>

      <Card style={cardStyle} styles={{ body: { padding: 8 } }}>
        <Table rowKey="id" columns={columns} dataSource={items} loading={loading} pagination={{ pageSize: 10, showSizeChanger: false }} />
      </Card>

      <Modal
        open={open}
        title={editing ? `编辑${title}` : `新增${title}`}
        onCancel={() => { setOpen(false); setEditing(null); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={saving}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ status: 'active' }}>
          <Form.Item label="编码" name="code" rules={[{ required: true, message: '请输入编码' }]}>
            <Input placeholder="请输入编码" />
          </Form.Item>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="请输入名称" />
          </Form.Item>
          {fields.map((field) => (
            <Form.Item key={field.name} label={field.label} name={field.name} rules={field.rules || []}>
              {field.input}
            </Form.Item>
          ))}
          <Form.Item label="状态" name="status">
            <Select options={statusOptions} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
