import React, { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Typography,
  Upload,
  message,
} from 'antd';
import { AlertOutlined, InboxOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { createDefect } from '../api';
import { useAlerts } from '../context/AlertContext';
import { defectTypes, severityOptions } from '../constants/defects';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Dragger } = Upload;

const cardStyle = {
  background: 'linear-gradient(180deg, rgba(11, 26, 40, 0.96) 0%, rgba(16, 34, 51, 0.92) 100%)',
  border: '1px solid rgba(102, 217, 255, 0.14)',
  borderRadius: 20,
  boxShadow: '0 18px 40px rgba(0, 0, 0, 0.22)',
};

export default function DefectCreatePage() {
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const { refreshAlerts } = useAlerts();
  const [fileList, setFileList] = useState([]);
  const [submitting, setSubmitting] = useState(false);
  const [criticalAlert, setCriticalAlert] = useState(null);

  const uploadProps = useMemo(() => ({
    multiple: false,
    maxCount: 1,
    accept: 'image/*',
    beforeUpload: () => false,
    fileList,
    onChange: ({ fileList: nextFileList }) => setFileList(nextFileList.slice(-1)),
    listType: 'picture',
  }), [fileList]);

  const handleSubmit = async (values) => {
    const formData = new FormData();
    formData.append('name', values.name);
    formData.append('defect_type', values.defect_type);
    formData.append('severity', values.severity);
    formData.append('location', values.location);
    formData.append('description', values.description || '');

    if (fileList[0]?.originFileObj) {
      formData.append('image', fileList[0].originFileObj);
    }

    setSubmitting(true);
    try {
      const created = await createDefect(formData);
      await refreshAlerts();
      message.success('新增缺陷成功');

      if (values.severity === 'critical') {
        setCriticalAlert({
          id: created.id,
          name: values.name,
          defect_type: values.defect_type,
          location: values.location,
        });
      } else {
        navigate('/defects');
      }
    } catch (error) {
      message.error('新增缺陷失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <Space direction="vertical" size={24} style={{ width: '100%' }}>
        <div>
          <Title level={2} style={{ color: '#f5fbff', margin: 0, fontFamily: 'Georgia, "Times New Roman", serif' }}>新增缺陷</Title>
          <Text style={{ color: 'rgba(216, 232, 246, 0.72)' }}>录入现场缺陷信息，并上传图像作为检测依据。</Text>
        </div>

        <Card style={cardStyle} styles={{ body: { padding: 24 } }}>
          <Form
            form={form}
            layout="vertical"
            initialValues={{ severity: 'medium' }}
            onFinish={handleSubmit}
          >
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 18 }}>
              <Form.Item label="标题" name="name" rules={[{ required: true, message: '请输入标题' }]}>
                <Input placeholder="例如：冲压面板边缘裂纹" />
              </Form.Item>

              <Form.Item label="类型" name="defect_type" rules={[{ required: true, message: '请选择类型' }]}>
                <Select placeholder="请选择缺陷类型" options={defectTypes.map((item) => ({ value: item, label: item }))} />
              </Form.Item>

              <Form.Item label="等级" name="severity" rules={[{ required: true, message: '请选择等级' }]}>
                <Select placeholder="请选择严重等级" options={severityOptions} />
              </Form.Item>

              <Form.Item label="位置" name="location" rules={[{ required: true, message: '请输入位置' }]}>
                <Input placeholder="例如：2号产线-A工位-外壳左侧" />
              </Form.Item>
            </div>

            <Form.Item label="描述" name="description">
              <TextArea rows={5} placeholder="请输入缺陷描述、发现背景、影响范围等信息" />
            </Form.Item>

            <Form.Item label="图片上传">
              <Dragger {...uploadProps} style={{ background: 'rgba(255,255,255,0.02)', borderColor: 'rgba(102, 217, 255, 0.3)', borderRadius: 18, padding: '18px 0' }}>
                <p className="ant-upload-drag-icon">
                  <InboxOutlined style={{ color: '#66d9ff' }} />
                </p>
                <p style={{ color: '#f0f7fd', fontSize: 16, marginBottom: 4 }}>拖拽图片到此处，或点击上传</p>
                <p style={{ color: 'rgba(216, 232, 246, 0.64)', margin: 0 }}>支持单张图片预览，提交时统一上传到后端</p>
              </Dragger>
            </Form.Item>

            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>提交缺陷</Button>
              <Button onClick={() => navigate('/defects')}>返回列表</Button>
            </Space>
          </Form>
        </Card>
      </Space>

      <Modal
        open={!!criticalAlert}
        footer={null}
        closable={false}
        centered
        width="100vw"
        styles={{
          content: {
            minHeight: '100vh',
            background: 'radial-gradient(circle at top, rgba(255,77,79,0.24), rgba(43,10,10,0.98) 42%, rgba(18,3,3,1) 100%)',
            borderRadius: 0,
            padding: 0,
          },
          body: { minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 32 },
          mask: { background: 'rgba(30,0,0,0.76)' },
        }}
      >
        {criticalAlert && (
          <div style={{ maxWidth: 720, width: '100%', textAlign: 'center', color: '#fff1f0' }}>
            <AlertOutlined style={{ fontSize: 72, color: '#ff7875', marginBottom: 20 }} />
            <div style={{ fontSize: 18, letterSpacing: 6, color: '#ffccc7', marginBottom: 12 }}>CRITICAL DEFECT ALERT</div>
            <Title level={1} style={{ color: '#fff1f0', marginBottom: 24, fontFamily: 'Georgia, "Times New Roman", serif' }}>
              严重缺陷已触发报警
            </Title>
            <div style={{ background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,120,117,0.25)', borderRadius: 24, padding: 28, textAlign: 'left' }}>
              <p><strong>缺陷标题：</strong>{criticalAlert.name}</p>
              <p><strong>缺陷类型：</strong>{criticalAlert.defect_type}</p>
              <p style={{ marginBottom: 0 }}><strong>缺陷位置：</strong>{criticalAlert.location}</p>
            </div>
            <Button
              type="primary"
              danger
              size="large"
              style={{ marginTop: 28, minWidth: 180, height: 48 }}
              onClick={() => {
                setCriticalAlert(null);
                navigate('/defects');
              }}
            >
              我已知晓
            </Button>
          </div>
        )}
      </Modal>
    </>
  );
}
