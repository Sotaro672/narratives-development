// frontend/console/production/src/presentation/pages/productionManagement.tsx

import React, { useEffect } from "react";
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

  // ===== rows をログ ==========
  useEffect(() => {
    console.log("[ProductionManagement] rows:", rows);
  }, [rows]);

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
            {/* Production ID */}
            <td className="text-blue-600 underline">{p.id}</td>

            {/* ★ ここを productBlueprintName に変更 */}
            <td>{(p as any).productBlueprintName || p.productBlueprintId}</td>

            {/* ★ 担当者名（あれば名前、なければID） */}
            <td>{(p as any).assigneeName || p.assigneeId}</td>

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
