import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Descriptions,
  Image,
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
import { useNavigate, useParams } from 'react-router-dom';
import {
  downloadFile,
  executeDefectAction,
  exportDefectDetailPdf,
  fetchDefect,
  fetchDefectFlowLogs,
  fetchUsers,
  getImageUrl,
} from '../api';
import { useAuth } from '../context/AuthContext';
import {
  defectActionLabels,
  severityColorMap,
  severityLabelMap,
  statusColorMap,
  statusLabelMap,
} from '../constants/defects';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

const cardStyle = {
  background: 'linear-gradient(180deg, rgba(11, 26, 40, 0.96) 0%, rgba(16, 34, 51, 0.92) 100%)',
  border: '1px solid rgba(102, 217, 255, 0.14)',
  borderRadius: 20,
  boxShadow: '0 18px 40px rgba(0, 0, 0, 0.22)',
};

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
  const { currentUser } = useAuth();
  const [detail, setDetail] = useState(null);
  const [logs, setLogs] = useState([]);
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [currentAction, setCurrentAction] = useState('');
  const [formState, setFormState] = useState({ assignee_id: undefined, acceptor_id: undefined, process_result: '', acceptance_comment: '', comment: '' });

  const loadDetail = async () => {
    const [detailResult, flowLogs, userList] = await Promise.all([
      fetchDefect(id),
      fetchDefectFlowLogs(id),
      fetchUsers(),
    ]);
    setDetail(detailResult);
    setLogs(flowLogs || []);
    setUsers(userList || []);
  };

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
  }, [id]);

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
            <Tag color="cyan">当前用户：{currentUser?.display_name || currentUser?.username}</Tag>
          </Space>
        </div>

        <Space wrap>
          <Button icon={<DownloadOutlined />} loading={exporting} onClick={handleExportPdf}>导出 PDF 报告</Button>
          {availableActions.map((action) => (
            <Button key={action} type={['accept', 'assign', 'start_process', 'submit_for_acceptance'].includes(action) ? 'primary' : 'default'} onClick={() => openActionModal(action)}>
              {defectActionLabels[action] || action}
            </Button>
          ))}
        </Space>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(320px, 1.4fr) minmax(280px, 1fr)', gap: 20 }}>
        <Card title={<span style={{ color: '#e8f3fb' }}>缺陷信息</span>} style={cardStyle} styles={{ header: { borderBottomColor: 'rgba(102, 217, 255, 0.12)' } }}>
          <Descriptions column={1} labelStyle={{ color: '#89a8bf', width: 132 }} contentStyle={{ color: '#f5fbff' }}>
            <Descriptions.Item label="标题">{detail.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="类型">{detail.defect_type || '-'}</Descriptions.Item>
            <Descriptions.Item label="类型ID">{detail.defect_type_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="等级">{severityLabelMap[detail.severity] || detail.severity || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">{statusLabelMap[detail.status] || detail.status || '-'}</Descriptions.Item>
            <Descriptions.Item label="所属产线">{detail.production_line || '-'}</Descriptions.Item>
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
              <Paragraph style={{ marginBottom: 0, color: '#f5fbff' }}>{detail.process_result || '暂无处理结果'}</Paragraph>
            </Descriptions.Item>
            <Descriptions.Item label="验收意见">
              <Paragraph style={{ marginBottom: 0, color: '#f5fbff' }}>{detail.acceptance_comment || '暂无验收意见'}</Paragraph>
            </Descriptions.Item>
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

      <Card title={<span style={{ color: '#e8f3fb' }}>闭环流转记录</span>} style={cardStyle} styles={{ header: { borderBottomColor: 'rgba(102, 217, 255, 0.12)' } }}>
        <Timeline
          items={logs.map((item) => ({
            color: statusColorMap[item.to_status] || 'blue',
            children: (
              <Space direction="vertical" size={4}>
                <Text style={{ color: '#f5fbff', fontWeight: 600 }}>{defectActionLabels[item.action] || item.action}</Text>
                <Text style={{ color: '#b9d4e6' }}>{item.operator_name} · {item.operator_role}</Text>
                <Text style={{ color: '#87a5bb' }}>{new Date(item.created_at).toLocaleString('zh-CN')}</Text>
                <Text style={{ color: '#dcebf7' }}>{statusLabelMap[item.from_status] || item.from_status} → {statusLabelMap[item.to_status] || item.to_status}</Text>
                {item.comment ? <Text style={{ color: '#f5fbff' }}>{item.comment}</Text> : null}
              </Space>
            ),
          }))}
        />
      </Card>

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
                <Text style={{ color: '#303133' }}>处理人</Text>
                <Select
                  style={{ width: '100%', marginTop: 8 }}
                  options={userOptions}
                  value={formState.assignee_id}
                  onChange={(value) => setFormState((prev) => ({ ...prev, assignee_id: value }))}
                />
              </div>
              <div>
                <Text style={{ color: '#303133' }}>验收人</Text>
                <Select
                  allowClear
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
              <Text style={{ color: '#303133' }}>处理结果</Text>
              <TextArea rows={4} value={formState.process_result} onChange={(event) => setFormState((prev) => ({ ...prev, process_result: event.target.value }))} style={{ marginTop: 8 }} />
            </div>
          ) : null}

          {actionNeedsAcceptanceComment(currentAction) ? (
            <div>
              <Text style={{ color: '#303133' }}>{currentAction === 'reject' ? '退回原因' : '验收意见'}</Text>
              <TextArea rows={4} value={formState.acceptance_comment} onChange={(event) => setFormState((prev) => ({ ...prev, acceptance_comment: event.target.value }))} style={{ marginTop: 8 }} />
            </div>
          ) : null}

          <div>
            <Text style={{ color: '#303133' }}>备注</Text>
            <TextArea rows={3} value={formState.comment} onChange={(event) => setFormState((prev) => ({ ...prev, comment: event.target.value }))} style={{ marginTop: 8 }} />
          </div>
        </Space>
      </Modal>
    </Space>
  );
}
