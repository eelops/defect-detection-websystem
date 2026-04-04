export const severityOptions = [
  { value: 'low', label: '低', color: 'green' },
  { value: 'medium', label: '中', color: 'blue' },
  { value: 'high', label: '高', color: 'orange' },
  { value: 'critical', label: '严重', color: 'red' },
];

export const statusOptions = [
  { value: 'open', label: '待分派', color: 'default' },
  { value: 'assigned', label: '已指派', color: 'blue' },
  { value: 'processing', label: '处理中', color: 'processing' },
  { value: 'pending_acceptance', label: '待验收', color: 'gold' },
  { value: 'accepted', label: '验收通过', color: 'success' },
  { value: 'rejected', label: '验收退回', color: 'volcano' },
  { value: 'closed', label: '已关闭', color: 'default' },
];

export const defectActionLabels = {
  assign: '指派',
  reassign: '改派',
  start_process: '开始处理',
  submit_for_acceptance: '提交验收',
  accept: '验收通过',
  reject: '验收退回',
  close: '关闭',
};

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
