import React from 'react';
import MasterDataManagePage from '../components/MasterDataManagePage';
import { createProductionLine, fetchProductionLines, updateProductionLine } from '../api';

export default function ProductionLineManagePage() {
  return (
    <MasterDataManagePage
      title="产线管理"
      subtitle="维护产线基础档案，作为设备与缺陷录入的上游主数据。"
      fields={[
        {
          name: 'description',
          label: '说明',
          input: <textarea className="ant-input" rows={4} placeholder="请输入产线说明" />,
          table: true,
          render: (value) => value || '-',
        },
      ]}
      fetchList={fetchProductionLines}
      createItem={createProductionLine}
      updateItem={updateProductionLine}
    />
  );
}
