import React, { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, Upload, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { createDefect, updateDefect } from '../api';

const { TextArea } = Input;

const defectTypes = ['裂纹', '划痕', '变形', '腐蚀', '其他'];
const severities = [
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' },
  { value: 'critical', label: '严重' },
];
const statuses = [
  { value: 'open', label: '待处理' },
  { value: 'fixing', label: '修复中' },
  { value: 'resolved', label: '已解决' },
  { value: 'closed', label: '已关闭' },
];

export default function DefectForm({ open, record, onClose, onSuccess }) {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [fileList, setFileList] = useState([]);
  const isEdit = !!record;

  useEffect(() => {
    if (open) {
      if (record) {
        form.setFieldsValue(record);
        setFileList([]);
      } else {
        form.resetFields();
        setFileList([]);
      }
    }
  }, [open, record, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      if (isEdit) {
        await updateDefect(record.id, values);
        message.success('更新成功');
      } else {
        const formData = new FormData();
        formData.append('name', values.name);
        formData.append('defect_type', values.defect_type);
        formData.append('severity', values.severity);
        if (values.description) formData.append('description', values.description);
        if (fileList.length > 0 && fileList[0].originFileObj) {
          formData.append('image', fileList[0].originFileObj);
        }
        await createDefect(formData);
        message.success('创建成功');
      }
      onSuccess();
    } catch (err) {
      if (err.errorFields) return; // form validation error
      message.error('操作失败');
    }
    setLoading(false);
  };

  return (
    <Modal
      title={isEdit ? '编辑缺陷' : '新增缺陷'}
      open={open}
      onOk={handleSubmit}
      onCancel={onClose}
      confirmLoading={loading}
      destroyOnClose
      width={520}
    >
      <Form form={form} layout="vertical" initialValues={{ severity: 'medium' }}>
        <Form.Item name="name" label="缺陷名称" rules={[{ required: true, message: '请输入缺陷名称' }]}>
          <Input placeholder="例如：表面裂纹 #001" />
        </Form.Item>
        <Form.Item name="defect_type" label="缺陷类型" rules={[{ required: true, message: '请选择缺陷类型' }]}>
          <Select placeholder="选择类型" options={defectTypes.map((t) => ({ value: t, label: t }))} />
        </Form.Item>
        <Form.Item name="severity" label="严重程度" rules={[{ required: true, message: '请选择严重程度' }]}>
          <Select options={severities} />
        </Form.Item>
        {isEdit && (
          <Form.Item name="status" label="状态">
            <Select options={statuses} />
          </Form.Item>
        )}
        <Form.Item name="description" label="描述">
          <TextArea rows={3} placeholder="请描述缺陷详情..." />
        </Form.Item>
        {!isEdit && (
          <Form.Item label="缺陷图片">
            <Upload
              listType="picture-card"
              fileList={fileList}
              onChange={({ fileList: fl }) => setFileList(fl.slice(-1))}
              beforeUpload={() => false}
              accept="image/*"
            >
              {fileList.length === 0 && (
                <div><PlusOutlined /><div style={{ marginTop: 8 }}>上传图片</div></div>
              )}
            </Upload>
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
}
