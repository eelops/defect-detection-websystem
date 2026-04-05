import React, { useEffect, useMemo, useState } from 'react';
import { Button, Card, DatePicker, Form, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd';
import { fetchOperationLogs } from '../api';

const { RangePicker } = DatePicker;
const { Title, Text } = Typography;

const cardStyle = {
  background: 'linear-gradient(180deg, rgba(11, 26, 40, 0.96) 0%, rgba(16, 34, 51, 0.92) 100%)',
  border: '1px solid rgba(102, 217, 255, 0.14)',
  borderRadius: 20,
  boxShadow: '0 18px 40px rgba(0, 0, 0, 0.22)',
};

const resultColorMap = {
  success: 'green',
  failure: 'red',
};

export default function OperationLogsPage() {
  const [form] = Form.useForm();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 });
  const [detailItem, setDetailItem] = useState(null);

  const loadLogs = async (page = pagination.current, pageSize = pagination.pageSize) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const params = {
        ...values,
        page,
        page_size: pageSize,
      };
      if (values.time_range?.[0] && values.time_range?.[1]) {
        params.start_time = values.time_range[0].toISOString();
        params.end_time = values.time_range[1].toISOString();
      }
      delete params.time_range;
      const data = await fetchOperationLogs(params);
      setItems(data.items || []);
      setPagination({ current: page, pageSize, total: data.total || 0 });
    } catch (error) {
      message.error(error?.response?.data?.detail || '操作日志加载失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadLogs(1, pagination.pageSize);
  }, []);

  const columns = useMemo(() => ([
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 180,
      render: (value) => value ? new Date(value).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作人',
      width: 150,
      render: (_, record) => record.actor_display_name || record.actor_username || record.actor_type || '-',
    },
    {
      title: '角色',
      dataIndex: 'actor_role',
      width: 120,
      render: (value) => value || '-',
    },
    { title: '模块', dataIndex: 'module', width: 120 },
    { title: '类型', dataIndex: 'action', width: 140 },
    {
      title: '结果',
      dataIndex: 'result',
      width: 100,
      render: (value) => <Tag color={resultColorMap[value] || 'default'}>{value}</Tag>,
    },
    { title: '对象', dataIndex: 'target_name', width: 180, render: (value) => value || '-' },
    { title: '摘要', dataIndex: 'summary' },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_, record) => <Button type="link" onClick={() => setDetailItem(record)}>查看详情</Button>,
    },
  ]), []);

  return (
    <Space direction="vertical" size={24} style={{ width: '100%' }}>
      <div>
        <Title level={2} style={{ color: '#f5fbff', margin: 0, fontFamily: 'Georgia, "Times New Roman", serif' }}>操作日志</Title>
        <Text style={{ color: 'rgba(216, 232, 246, 0.72)' }}>查看关键业务操作和安全审计记录。</Text>
      </div>

      <Card style={cardStyle}>
        <Form form={form} layout="inline" onFinish={() => loadLogs(1, pagination.pageSize)}>
          <Form.Item name="module" label="模块">
            <Select allowClear style={{ width: 140 }} options={[
              { value: 'auth', label: '认证' },
              { value: 'user', label: '用户' },
              { value: 'users', label: '用户' },
              { value: 'defect', label: '缺陷' },
              { value: 'defects', label: '缺陷' },
              { value: 'external', label: '外部' },
            ]} />
          </Form.Item>
          <Form.Item name="result" label="结果">
            <Select allowClear style={{ width: 120 }} options={[{ value: 'success', label: '成功' }, { value: 'failure', label: '失败' }]} />
          </Form.Item>
          <Form.Item name="actor_username" label="操作人">
            <Input placeholder="用户名" />
          </Form.Item>
          <Form.Item name="keyword" label="关键词">
            <Input placeholder="摘要/对象" />
          </Form.Item>
          <Form.Item name="time_range" label="时间范围">
            <RangePicker showTime />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">查询</Button>
              <Button onClick={() => { form.resetFields(); loadLogs(1, pagination.pageSize); }}>重置</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Card style={cardStyle} styles={{ body: { padding: 8 } }}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={items}
          loading={loading}
          scroll={{ x: 1280 }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            onChange: (page, pageSize) => loadLogs(page, pageSize),
          }}
        />
      </Card>

      <Modal open={!!detailItem} title="日志详情" footer={null} onCancel={() => setDetailItem(null)} width={760} destroyOnHidden>
        {detailItem && (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Text style={{ color: '#f3fbff' }}>摘要：{detailItem.summary}</Text>
            <Text style={{ color: '#dbe8f3' }}>请求：{detailItem.request_method || '-'} {detailItem.request_path || '-'}</Text>
            <Text style={{ color: '#dbe8f3' }}>IP：{detailItem.ip_address || '-'}</Text>
            <Text style={{ color: '#dbe8f3' }}>User-Agent：{detailItem.user_agent || '-'}</Text>
            <pre style={{ margin: 0, padding: 12, background: '#081018', color: '#d6e8f5', borderRadius: 12, overflow: 'auto', border: '1px solid rgba(102, 217, 255, 0.14)' }}>{JSON.stringify(detailItem.details || {}, null, 2)}</pre>
          </Space>
        )}
      </Modal>
    </Space>
  );
}
