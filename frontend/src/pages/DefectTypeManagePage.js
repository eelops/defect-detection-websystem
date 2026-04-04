import React from 'react';
import { Input, Select } from 'antd';
import MasterDataManagePage from '../components/MasterDataManagePage';
import { createDefectType, fetchDefectTypes, updateDefectType } from '../api';
import { severityOptions } from '../constants/defects';

export default function DefectTypeManagePage() {
  return (
    <MasterDataManagePage
      title="缺陷类型管理"
      subtitle="维护缺陷分类标准，供缺陷录入、筛选与统计统一使用。"
      fields={[
        {
          name: 'severity',
          label: '默认等级',
          input: <Select allowClear placeholder="请选择默认等级" options={severityOptions} />,
          table: true,
          render: (value) => severityOptions.find((item) => item.value === value)?.label || '-',
        },
        {
          name: 'description',
          label: '说明',
          input: <Input.TextArea rows={4} placeholder="请输入缺陷类型说明" />,
        },
      ]}
      fetchList={fetchDefectTypes}
      createItem={createDefectType}
      updateItem={updateDefectType}
    />
  );
}
