import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { createUser, fetchUsers, updateUser } from '../api';

const { Title, Text } = Typography;

const roleOptions = [
  { value: 'admin', label: '管理员' },
  { value: 'inspector', label: '质检员' },
  { value: 'viewer', label: '查看员' },
];

const cardStyle = {
  background: 'linear-gradient(180deg, rgba(11, 26, 40, 0.96) 0%, rgba(16, 34, 51, 0.92) 100%)',
  border: '1px solid rgba(102, 217, 255, 0.14)',
  borderRadius: 20,
  boxShadow: '0 18px 40px rgba(0, 0, 0, 0.22)',
};

export default function UserManagementPage() {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [editingUser, setEditingUser] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();

  const loadUsers = async () => {
    setLoading(true);
    try {
      const data = await fetchUsers();
      setItems(data || []);
    } catch (error) {
      message.error(error?.response?.data?.detail || '用户列表加载失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadUsers();
  }, []);

  const handleCreate = () => {
    setEditingUser(null);
    form.resetFields();
    form.setFieldsValue({ role: 'viewer', is_active: true });
    setOpen(true);
  };

  const handleEdit = (record) => {
    setEditingUser(record);
    form.setFieldsValue({
      display_name: record.display_name,
      role: record.role,
      is_active: record.is_active,
      password: '',
    });
    setOpen(true);
  };

  const handleFinish = async (values) => {
    setSubmitting(true);
    try {
      if (editingUser) {
        const payload = {
          display_name: values.display_name,
          role: values.role,
          is_active: values.is_active,
        };
        if (values.password) {
          payload.password = values.password;
        }
        await updateUser(editingUser.id, payload);
        message.success('用户更新成功');
      } else {
        await createUser(values);
        message.success('用户创建成功');
      }
      setOpen(false);
      form.resetFields();
      loadUsers();
    } catch (error) {
      message.error(error?.response?.data?.detail || '用户保存失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggleActive = async (record, isActive) => {
    try {
      await updateUser(record.id, { is_active: isActive });
      message.success(isActive ? '用户已启用' : '用户已禁用');
      loadUsers();
    } catch (error) {
      message.error(error?.response?.data?.detail || '状态更新失败');
    }
  };

  const columns = useMemo(() => ([
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '用户名', dataIndex: 'username', width: 140 },
    { title: '显示名称', dataIndex: 'display_name', width: 160 },
    {
      title: '角色',
      dataIndex: 'role_label',
      width: 120,
      render: (_, record) => <Tag color={record.role === 'admin' ? 'red' : record.role === 'inspector' ? 'blue' : 'default'}>{record.role_label}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      width: 120,
      render: (value) => <Tag color={value ? 'green' : 'default'}>{value ? '启用' : '禁用'}</Tag>,
    },
    {
      title: '最近登录',
      dataIndex: 'last_login_at',
      width: 180,
      render: (value) => value ? new Date(value).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'actions',
      width: 220,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" onClick={() => handleEdit(record)}>编辑</Button>
          <Popconfirm
            title={record.is_active ? '确认禁用该用户吗？' : '确认启用该用户吗？'}
            onConfirm={() => handleToggleActive(record, !record.is_active)}
          >
            <Button type="link">{record.is_active ? '禁用' : '启用'}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]), []);

  return (
    <Space direction="vertical" size={24} style={{ width: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', gap: 16, flexWrap: 'wrap' }}>
        <div>
          <Title level={2} style={{ color: '#f5fbff', margin: 0, fontFamily: 'Georgia, "Times New Roman", serif' }}>用户管理</Title>
          <Text style={{ color: 'rgba(216, 232, 246, 0.72)' }}>维护系统账号、角色权限和启用状态。</Text>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新增用户</Button>
      </div>

      <Card style={cardStyle} styles={{ body: { padding: 8 } }}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={items}
          loading={loading}
          scroll={{ x: 980 }}
          pagination={false}
        />
      </Card>

      <Modal
        open={open}
        title={editingUser ? '编辑用户' : '新增用户'}
        onCancel={() => setOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={submitting}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" onFinish={handleFinish}>
          {!editingUser && (
            <>
              <Form.Item label="用户名" name="username" rules={[{ required: true, message: '请输入用户名' }]}>
                <Input placeholder="请输入用户名" />
              </Form.Item>
              <Form.Item label="密码" name="password" rules={[{ required: true, message: '请输入密码' }]}>
                <Input.Password placeholder="请输入初始密码" />
              </Form.Item>
            </>
          )}
          {editingUser && (
            <Form.Item label="重置密码" name="password">
              <Input.Password placeholder="留空表示不修改密码" />
            </Form.Item>
          )}
          <Form.Item label="显示名称" name="display_name" rules={[{ required: true, message: '请输入显示名称' }]}>
            <Input placeholder="请输入显示名称" />
          </Form.Item>
          <Form.Item label="角色" name="role" rules={[{ required: true, message: '请选择角色' }]}>
            <Select options={roleOptions} placeholder="请选择角色" />
          </Form.Item>
          <Form.Item label="启用状态" name="is_active" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
