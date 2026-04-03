export const defectTypes = ['裂纹', '划痕', '变形', '腐蚀', '其他'];

export const severityOptions = [
  { value: 'low', label: '低', color: 'green' },
  { value: 'medium', label: '中', color: 'blue' },
  { value: 'high', label: '高', color: 'orange' },
  { value: 'critical', label: '严重', color: 'red' },
];

export const statusOptions = [
  { value: 'open', label: '待处理', color: 'default' },
  { value: 'fixing', label: '修复中', color: 'processing' },
  { value: 'resolved', label: '已解决', color: 'success' },
  { value: 'closed', label: '已关闭', color: 'default' },
];

export const severityLabelMap = Object.fromEntries(
  severityOptions.map((item) => [item.value, item.label])
);

export const severityColorMap = Object.fromEntries(
  severityOptions.map((item) => [item.value, item.color])
);

export const statusLabelMap = Object.fromEntries(
  statusOptions.map((item) => [item.value, item.label])
);

export const statusColorMap = Object.fromEntries(
  statusOptions.map((item) => [item.value, item.color])
);
