import React, { useEffect, useState } from 'react';
import { Select, Input } from 'antd';
import MasterDataManagePage from '../components/MasterDataManagePage';
import { createEquipment, fetchEquipment, fetchProductionLines, updateEquipment } from '../api';

export default function EquipmentManagePage() {
  const [lineOptions, setLineOptions] = useState([]);

  useEffect(() => {
    fetchProductionLines({ status: 'active' }).then((result) => {
      setLineOptions((result.items || []).map((item) => ({ value: item.id, label: item.name })));
    });
  }, []);

  return (
    <MasterDataManagePage
      title="设备管理"
      subtitle="维护设备基础资料，并绑定所属产线。"
      fields={[
        {
          name: 'production_line_id',
          label: '所属产线',
          input: <Select placeholder="请选择所属产线" options={lineOptions} />,
          rules: [{ required: true, message: '请选择所属产线' }],
          table: true,
          render: (_, record) => record.production_line_name || '-',
        },
        {
          name: 'model',
          label: '设备型号',
          input: <Input placeholder="请输入设备型号" />,
          table: true,
          render: (value) => value || '-',
        },
        {
          name: 'description',
          label: '说明',
          input: <textarea className="ant-input" rows={4} placeholder="请输入设备说明" />,
        },
      ]}
      fetchList={fetchEquipment}
      createItem={createEquipment}
      updateItem={updateEquipment}
    />
  );
}
