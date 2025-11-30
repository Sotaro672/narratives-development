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
            {/* ★ productBlueprintName（またはID） */}
            <td>{(p as any).productBlueprintName || p.productBlueprintId}</td>

            {/* ★ ブランド名（あればブランド名、なければ空文字） */}
            <td>{(p as any).brandName || ""}</td>

            {/* ★ 担当者名（名前があれば名前、なければID） */}
            <td>{(p as any).assigneeName || p.assigneeId}</td>

            {/* ステータス */}
            <td>{p.status}</td>

            {/* 合計数量 */}
            <td>{p.totalQuantity}</td>

            {/* 印刷日時ラベル */}
            <td>{p.printedAtLabel}</td>

            {/* 作成日時ラベル */}
            <td>{p.createdAtLabel}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
