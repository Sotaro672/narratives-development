// frontend/console/production/src/presentation/pages/productionManagement.tsx

import React from "react";
import { useProductionManagement } from "../hook/useProductionManagement";
import List from "../../../../shell/src/layout/List/List";

export default function ProductionManagement() {
  const {
    headers,
    rows,
    handleCreate,
    handleReset,
    handleRowClick,
  } = useProductionManagement();

  return (
    <div className="p-0">
      <List
        title="商品生産"
        headerCells={headers}
        showCreateButton
        createLabel="生産計画を作成"
        showResetButton
        onCreate={handleCreate}
        onReset={handleReset}
      >
        {rows.map((p) => (
          <tr
            key={p.id}
            className="cursor-pointer hover:bg-blue-50 transition-colors"
            onClick={() => handleRowClick(p.id)}
          >
            <td className="text-blue-600 underline">{p.id}</td>
            <td>{p.productBlueprintId}</td>
            <td>{p.assigneeId}</td>
            <td>{p.status}</td>
            <td>{p.totalQuantity}</td>
            <td>{p.printedAtLabel}</td>
            <td>{p.createdAtLabel}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
