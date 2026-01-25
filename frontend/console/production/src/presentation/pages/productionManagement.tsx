// frontend/console/production/src/presentation/pages/productionManagement.tsx

import { useProductionManagement } from "../hook/useProductionManagement";
import List from "../../../../shell/src/layout/List/List";

function formatProductionStatusJa(status: unknown): string {
  switch (status) {
    case "planned":
      return "計画中";
    case "printed":
      return "印刷済み";
    default:
      return typeof status === "string" ? status : "";
  }
}

export default function ProductionManagement() {
  const { headers, rows, handleCreate, handleReset, handleRowClick } =
    useProductionManagement();

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
            {/* ★ プロダクト名 */}
            <td>{(p as any).productName || p.productBlueprintId}</td>

            {/* ★ ブランド名 */}
            <td>{(p as any).brandName || ""}</td>

            {/* ★ 担当者名（あれば名前、なければID） */}
            <td>{(p as any).assigneeName || p.assigneeId}</td>

            {/* ステータス（日本語表示） */}
            <td>{formatProductionStatusJa((p as any).status)}</td>

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
