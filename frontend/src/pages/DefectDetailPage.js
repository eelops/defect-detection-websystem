import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Descriptions,
  Image,
  Form,
  Input,
  Modal,
  Select,
  Skeleton,
  Space,
  Tag,
  Timeline,
  Typography,
  message,
} from 'antd';
import { ArrowLeftOutlined, DownloadOutlined } from '@ant-design/icons';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  updateDefect,
  downloadFile,
  executeDefectAction,
  exportDefectDetailPdf,
  fetchAssignableUsers,
  fetchDefect,
  fetchDefectFlowLogs,
  getImageUrl,
} from '../api';
import { useAuth } from '../context/AuthContext';
import {
  defectActionLabels,
  severityOptions,
  severityColorMap,
  severityLabelMap,
  statusColorMap,
  statusLabelMap,
} from '../constants/defects';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

const buildPdfFileName = (detail, id) => {
  const baseName = (detail?.name || `缺陷_${id}`).replace(/[\\/:*?"<>|]/g, '_');
  return `${baseName}_详情报告.pdf`;
};

const actionNeedsAssignee = (action) => ['assign', 'reassign'].includes(action);
const actionNeedsProcessResult = (action) => action === 'submit_for_acceptance';
const actionNeedsAcceptanceComment = (action) => ['accept', 'reject'].includes(action);

export default function DefectDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { currentUser, hasRole } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();
  const [editForm] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const editOpen = searchParams.get('edit') === '1' && hasRole('admin', 'inspector');
  const closeEdit = () => setSearchParams({}, { replace: true });
  const [detail, setDetail] = useState(null);
  const [logs, setLogs] = useState([]);
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [currentAction, setCurrentAction] = useState('');
  const [formState, setFormState] = useState({ assignee_id: undefined, acceptor_id: undefined, process_result: '', acceptance_comment: '', comment: '' });

  const loadDetail = useCallback(async () => {
    const [detailResult, flowLogs, userList] = await Promise.all([
      fetchDefect(id),
      fetchDefectFlowLogs(id),
      fetchAssignableUsers(),
    ]);
    setDetail(detailResult);
    setLogs(flowLogs || []);
    setUsers(userList || []);
  }, [id]);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        await loadDetail();
      } catch (error) {
        message.error('缺陷详情加载失败');
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [loadDetail]);

  useEffect(() => {
    if (editOpen && detail) {
      editForm.setFieldsValue({ name: detail.name, severity: detail.severity, description: detail.description || '' });
    }
  }, [editOpen, detail, editForm]);

  const saveDetails = async (values) => {
    setSaving(true);
    try {
      await updateDefect(id, values);
      closeEdit();
      await loadDetail();
      message.success('缺陷信息已更新');
    } catch (error) {
      message.error(error?.response?.data?.detail || '保存失败');
    } finally { setSaving(false); }
  };

  const handleExportPdf = async () => {
    setExporting(true);
    try {
      const blob = await exportDefectDetailPdf(id);
      downloadFile(blob, buildPdfFileName(detail, id));
      message.success('PDF 报告导出成功');
    } catch (error) {
      message.error(error?.response?.data?.detail || 'PDF 报告导出失败');
    } finally {
      setExporting(false);
    }
  };

  const openActionModal = (action) => {
    setCurrentAction(action);
    setFormState({
      assignee_id: detail?.assignee_id,
      acceptor_id: detail?.acceptor_id,
      process_result: detail?.process_result || '',
      acceptance_comment: '',
      comment: '',
    });
    setModalOpen(true);
  };

  const handleSubmitAction = async () => {
    setSubmitting(true);
    try {
      await executeDefectAction(id, {
        action: currentAction,
        assignee_id: formState.assignee_id,
        acceptor_id: formState.acceptor_id,
        process_result: formState.process_result || undefined,
        acceptance_comment: formState.acceptance_comment || undefined,
        comment: formState.comment || undefined,
      });
      setModalOpen(false);
      await loadDetail();
      message.success(`${defectActionLabels[currentAction] || currentAction}成功`);
    } catch (error) {
      message.error(error?.response?.data?.detail || '流程操作失败');
    } finally {
      setSubmitting(false);
    }
  };

  const availableActions = detail?.available_actions || [];
  const userOptions = useMemo(() => users.map((item) => ({ value: item.id, label: `${item.display_name}（${item.role_label}）` })), [users]);

  if (loading) {
    return <Skeleton active paragraph={{ rows: 10 }} />;
  }

  if (!detail) {
    return (
      <Card className="content-card">
        <Text>未找到对应缺陷。</Text>
      </Card>
    );
  }

  return (
    <Space direction="vertical" size={20} className="page-stack">
      <div className="page-header" style={{ alignItems: 'flex-start' }}>
        <div className="page-heading">
          <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => navigate('/defects')} style={{ paddingLeft: 0 }}>
            返回列表
          </Button>
          <Title level={2} className="page-heading-title">{detail.name || `缺陷 #${id}`}</Title>
          <Space size={12} style={{ marginTop: 10 }}>
            <Tag color={severityColorMap[detail.severity]}>{severityLabelMap[detail.severity] || detail.severity}</Tag>
            <Tag color={statusColorMap[detail.status]}>{statusLabelMap[detail.status] || detail.status}</Tag>
            <Tag color="blue">当前用户：{currentUser?.display_name || currentUser?.username}</Tag>
          </Space>
        </div>

        <Space wrap>
          {hasRole('admin', 'inspector') && <Button onClick={() => setSearchParams({ edit: '1' })}>编辑信息</Button>}
          <Button icon={<DownloadOutlined />} loading={exporting} onClick={handleExportPdf}>导出 PDF 报告</Button>
          {availableActions.map((action) => (
            <Button key={action} type={['accept', 'assign', 'start_process', 'submit_for_acceptance'].includes(action) ? 'primary' : 'default'} onClick={() => openActionModal(action)}>
              {defectActionLabels[action] || action}
            </Button>
          ))}
        </Space>
      </div>

      <div className="detail-grid">
        <Card title="缺陷信息" className="content-card">
          <Descriptions column={1} labelStyle={{ width: 132 }}>
            <Descriptions.Item label="标题">{detail.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="类型">{detail.defect_type || '-'}</Descriptions.Item>
            <Descriptions.Item label="类型ID">{detail.defect_type_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="等级">{severityLabelMap[detail.severity] || detail.severity || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">{statusLabelMap[detail.status] || detail.status || '-'}</Descriptions.Item>
            <Descriptions.Item label="所属产线">{detail.production_line_name || '-'}</Descriptions.Item>
            <Descriptions.Item label="产线ID">{detail.production_line_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="设备">{detail.equipment_name || '-'}</Descriptions.Item>
            <Descriptions.Item label="设备ID">{detail.equipment_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="位置">{detail.location || '-'}</Descriptions.Item>
            <Descriptions.Item label="处理人">{detail.assignee_name || '-'}</Descriptions.Item>
            <Descriptions.Item label="验收人">{detail.acceptor_name || '-'}</Descriptions.Item>
            <Descriptions.Item label="指派时间">{detail.assigned_at ? new Date(detail.assigned_at).toLocaleString('zh-CN') : '-'}</Descriptions.Item>
            <Descriptions.Item label="提交验收时间">{detail.processed_at ? new Date(detail.processed_at).toLocaleString('zh-CN') : '-'}</Descriptions.Item>
            <Descriptions.Item label="验收时间">{detail.accepted_at ? new Date(detail.accepted_at).toLocaleString('zh-CN') : '-'}</Descriptions.Item>
            <Descriptions.Item label="关闭时间">{detail.closed_at ? new Date(detail.closed_at).toLocaleString('zh-CN') : '-'}</Descriptions.Item>
            <Descriptions.Item label="处理结果">
              <Paragraph style={{ marginBottom: 0 }}>{detail.process_result || '暂无处理结果'}</Paragraph>
            </Descriptions.Item>
            <Descriptions.Item label="验收意见">
              <Paragraph style={{ marginBottom: 0 }}>{detail.acceptance_comment || '暂无验收意见'}</Paragraph>
            </Descriptions.Item>
            <Descriptions.Item label="描述">
              <Paragraph style={{ marginBottom: 0 }}>{detail.description || '暂无描述'}</Paragraph>
            </Descriptions.Item>
          </Descriptions>
        </Card>

        <Card title="缺陷图片" className="content-card">
          {detail.image_path ? (
            <Image src={getImageUrl(detail.image_path)} style={{ width: '100%', borderRadius: 8 }} />
          ) : (
            <div className="image-placeholder">暂无图片</div>
          )}
        </Card>
      </div>

      <Card title="闭环流转记录" className="content-card">
        <Timeline
          items={logs.map((item) => ({
            color: statusColorMap[item.to_status] || 'blue',
            children: (
              <div className="timeline-entry">
                <Text strong>{defectActionLabels[item.action] || item.action}</Text>
                <Text type="secondary">{item.operator_name} · {item.operator_role}</Text>
                <Text type="secondary">{new Date(item.created_at).toLocaleString('zh-CN')}</Text>
                <Text>{statusLabelMap[item.from_status] || item.from_status} → {statusLabelMap[item.to_status] || item.to_status}</Text>
                {item.comment ? <Text>{item.comment}</Text> : null}
              </div>
            ),
          }))}
        />
      </Card>

      <Modal open={editOpen} title="编辑缺陷信息" onCancel={closeEdit} onOk={() => editForm.submit()} confirmLoading={saving} destroyOnHidden>
        <Form form={editForm} layout="vertical" onFinish={saveDetails}>
          <Form.Item name="name" label="标题" rules={[{ required: true, whitespace: true, message: '请输入标题' }]}><Input /></Form.Item>
          <Form.Item name="severity" label="等级" rules={[{ required: true, message: '请选择等级' }]}><Select options={severityOptions} /></Form.Item>
          <Form.Item name="description" label="描述"><TextArea rows={4} /></Form.Item>
        </Form>
      </Modal>

      <Modal
        open={modalOpen}
        title={defectActionLabels[currentAction] || '流程操作'}
        onCancel={() => setModalOpen(false)}
        onOk={handleSubmitAction}
        confirmLoading={submitting}
        destroyOnHidden
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          {actionNeedsAssignee(currentAction) ? (
            <>
              <div>
                <Text>处理人</Text>
                <Select
                  style={{ width: '100%', marginTop: 8 }}
                  options={userOptions}
                  value={formState.assignee_id}
                  onChange={(value) => setFormState((prev) => ({ ...prev, assignee_id: value }))}
                />
              </div>
              <div>
                <Text>验收人</Text>
                <Select
                  style={{ width: '100%', marginTop: 8 }}
                  options={userOptions}
                  value={formState.acceptor_id}
                  onChange={(value) => setFormState((prev) => ({ ...prev, acceptor_id: value }))}
                />
              </div>
            </>
          ) : null}

          {actionNeedsProcessResult(currentAction) ? (
            <div>
              <Text>处理结果</Text>
              <TextArea rows={4} value={formState.process_result} onChange={(event) => setFormState((prev) => ({ ...prev, process_result: event.target.value }))} style={{ marginTop: 8 }} />
            </div>
          ) : null}

          {actionNeedsAcceptanceComment(currentAction) ? (
            <div>
              <Text>{currentAction === 'reject' ? '退回原因' : '验收意见'}</Text>
              <TextArea rows={4} value={formState.acceptance_comment} onChange={(event) => setFormState((prev) => ({ ...prev, acceptance_comment: event.target.value }))} style={{ marginTop: 8 }} />
            </div>
          ) : null}

          <div>
            <Text>备注</Text>
            <TextArea rows={3} value={formState.comment} onChange={(event) => setFormState((prev) => ({ ...prev, comment: event.target.value }))} style={{ marginTop: 8 }} />
          </div>
        </Space>
      </Modal>
    </Space>
  );
}
