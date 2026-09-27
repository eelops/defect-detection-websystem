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
import { ApiOutlined, DeleteOutlined, DownloadOutlined, EditOutlined, EyeOutlined, ReloadOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import {
  API_BASE_URL,
  deleteDefect,
  downloadFile,
  exportDefectsExcel,
  fetchDefectTypes,
  fetchDefects,
  fetchEquipment,
  fetchProductionLines,
  getImageUrl,
} from '../api';
import { useAuth } from '../context/AuthContext';
import {
  severityColorMap,
  severityLabelMap,
  severityOptions,
  statusColorMap,
  statusLabelMap,
  statusOptions,
} from '../constants/defects';

const { Title, Text } = Typography;

const buildExcelFileName = () => {
  const today = new Date();
  const yyyy = today.getFullYear();
  const mm = String(today.getMonth() + 1).padStart(2, '0');
  const dd = String(today.getDate()).padStart(2, '0');
  return `缺陷台账报表_${yyyy}-${mm}-${dd}.xlsx`;
};

export default function DefectListPage() {
  const navigate = useNavigate();
  const { hasRole } = useAuth();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [masterDataLoading, setMasterDataLoading] = useState(false);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [productionLines, setProductionLines] = useState([]);
  const [equipmentList, setEquipmentList] = useState([]);
  const [defectTypes, setDefectTypes] = useState([]);
  const [filters, setFilters] = useState({
    defect_type_id: undefined,
    production_line_id: undefined,
    equipment_id: undefined,
    severity: undefined,
    status: undefined,
  });
  const currentPage = pagination.current;
  const currentPageSize = pagination.pageSize;

  const canEdit = hasRole('admin', 'inspector');
  const canDelete = hasRole('admin');

  useEffect(() => {
    const loadMasterData = async () => {
      setMasterDataLoading(true);
      try {
        const [lineResult, equipmentResult, defectTypeResult] = await Promise.all([
          fetchProductionLines({ status: 'active' }),
          fetchEquipment({ status: 'active' }),
          fetchDefectTypes({ status: 'active' }),
        ]);
        setProductionLines(lineResult.items || []);
        setEquipmentList(equipmentResult.items || []);
        setDefectTypes(defectTypeResult.items || []);
      } catch (error) {
        message.error('基础数据加载失败');
      } finally {
        setMasterDataLoading(false);
      }
    };

    loadMasterData();
  }, []);

  const loadData = useCallback(async (page = 1, pageSize = 10) => {
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
  }, [filters]);

  useEffect(() => {
    loadData(1, currentPageSize);
  }, [filters, loadData, currentPageSize]);

  const handleDelete = useCallback(async (id) => {
    try {
      await deleteDefect(id);
      message.success('删除成功');
      loadData(currentPage, currentPageSize);
    } catch (error) {
      message.error(error?.response?.data?.detail || '删除失败');
    }
  }, [currentPage, currentPageSize, loadData]);

  const handleExportExcel = async () => {
    setExporting(true);
    try {
      const blob = await exportDefectsExcel(filters);
      downloadFile(blob, buildExcelFileName());
      message.success('Excel 报表导出成功');
    } catch (error) {
      message.error(error?.response?.data?.detail || 'Excel 报表导出失败');
    } finally {
      setExporting(false);
    }
  };

  const productionLineOptions = useMemo(
    () => productionLines.map((item) => ({ value: item.id, label: `${item.code} · ${item.name}` })),
    [productionLines]
  );

  const defectTypeOptions = useMemo(
    () => defectTypes.map((item) => ({ value: item.id, label: `${item.code} · ${item.name}` })),
    [defectTypes]
  );

  const equipmentOptions = useMemo(() => {
    const filteredItems = filters.production_line_id
      ? equipmentList.filter((item) => item.production_line_id === filters.production_line_id)
      : equipmentList;

    return filteredItems.map((item) => ({ value: item.id, label: `${item.code} · ${item.name}` }));
  }, [equipmentList, filters.production_line_id]);

  const columns = useMemo(() => {
    const actionWidth = canDelete ? 260 : 180;
    return [
      { title: 'ID', dataIndex: 'id', width: 80 },
      {
        title: '图片',
        dataIndex: 'image_path',
        width: 110,
        render: (value) => value ? <Image width={64} height={64} src={getImageUrl(value)} style={{ objectFit: 'cover', borderRadius: 8 }} /> : <Text type="secondary">暂无</Text>,
      },
      { title: '标题', dataIndex: 'name', ellipsis: true, width: 180 },
      { title: '缺陷类型', dataIndex: 'defect_type', width: 160, render: (value) => value || '-' },
      { title: '所属产线', dataIndex: 'production_line_name', width: 140, render: (value) => value || '-' },
      { title: '设备', dataIndex: 'equipment_name', width: 140, render: (value) => value || '-' },
      {
        title: '等级',
        dataIndex: 'severity',
        width: 100,
        render: (value) => <Tag color={severityColorMap[value]}>{severityLabelMap[value] || value}</Tag>,
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 130,
        render: (value) => <Tag color={statusColorMap[value]}>{statusLabelMap[value] || value}</Tag>,
      },
      {
        title: '处理人',
        dataIndex: 'assignee_name',
        width: 120,
        render: (value) => value || '-',
      },
      {
        title: '验收人',
        dataIndex: 'acceptor_name',
        width: 120,
        render: (value) => value || '-',
      },
      {
        title: '位置',
        dataIndex: 'location',
        width: 180,
        render: (value) => value || '-',
      },
      {
        title: '更新时间',
        dataIndex: 'updated_at',
        width: 180,
        render: (value) => value ? new Date(value).toLocaleString('zh-CN') : '-',
      },
      {
        title: '操作',
        key: 'actions',
        width: actionWidth,
        fixed: 'right',
        render: (_, record) => (
          <Space size="small" wrap>
            <Button type="link" icon={<EyeOutlined />} onClick={() => navigate(`/defects/${record.id}`)}>查看详情</Button>
            {canEdit && <Button type="link" icon={<EditOutlined />} onClick={() => navigate(`/defects/${record.id}?edit=1`)}>编辑</Button>}
            {canDelete && (
              <Popconfirm title="确认删除该缺陷吗？" onConfirm={() => handleDelete(record.id)}>
                <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
              </Popconfirm>
            )}
          </Space>
        ),
      },
    ];
  }, [canDelete, canEdit, handleDelete, navigate]);

  return (
    <Space direction="vertical" size={20} className="page-stack">
      <div className="page-header">
        <div className="page-heading">
          <Title level={2} className="page-heading-title">缺陷列表</Title>
          <Text className="page-heading-subtitle">按主数据、等级、状态筛选并维护缺陷闭环台账。</Text>
        </div>
        <Space>
          <Button icon={<DownloadOutlined />} loading={exporting} onClick={handleExportExcel}>导出 Excel</Button>
          {canEdit && <Button type="primary" onClick={() => navigate('/defects/new')}>新增缺陷</Button>}
        </Space>
      </div>

      <Alert
        type="info"
        showIcon
        icon={<ApiOutlined />}
        message="第三方缺陷 JSON 推送接口"
        description={(
          <div style={{ lineHeight: 1.8 }}>
            <div><strong>接口地址：</strong><code>POST {API_BASE_URL}/api/external/report</code></div>
            <div><strong>认证方式：</strong>Header 携带 <code>X-API-Key</code></div>
            <div><strong>数据格式：</strong>application/json，支持图片 base64、缺陷类型、等级、位置、置信度、来源等字段</div>
          </div>
        )}
      />

      <Card className="toolbar-card">
        <div className="filter-grid">
          <div className="filter-controls">
            <Select
              allowClear
              placeholder="按缺陷类型筛选"
              style={{ width: 180 }}
              options={defectTypeOptions}
              loading={masterDataLoading}
              showSearch
              optionFilterProp="label"
              onChange={(value) => setFilters((prev) => ({ ...prev, defect_type_id: value }))}
            />
            <Select
              allowClear
              placeholder="按产线筛选"
              style={{ width: 180 }}
              options={productionLineOptions}
              loading={masterDataLoading}
              showSearch
              optionFilterProp="label"
              onChange={(value) => setFilters((prev) => ({ ...prev, production_line_id: value, equipment_id: undefined }))}
            />
            <Select
              allowClear
              placeholder="按设备筛选"
              style={{ width: 180 }}
              options={equipmentOptions}
              loading={masterDataLoading}
              showSearch
              optionFilterProp="label"
              onChange={(value) => setFilters((prev) => ({ ...prev, equipment_id: value }))}
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
              style={{ width: 180 }}
              options={statusOptions}
              onChange={(value) => setFilters((prev) => ({ ...prev, status: value }))}
            />
          </div>

          <Button icon={<ReloadOutlined />} onClick={() => loadData(1, currentPageSize)}>刷新</Button>
        </div>
      </Card>

      <Card className="content-card table-card">
        <Table
          rowKey="id"
          columns={columns}
          dataSource={items}
          loading={loading}
          scroll={{ x: 1940 }}
          pagination={{
            current: pagination.current,
            pageSize: currentPageSize,
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
