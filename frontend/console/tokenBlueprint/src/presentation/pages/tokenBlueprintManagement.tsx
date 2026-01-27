// frontend/console/tokenBlueprint/src/presentation/pages/tokenBlueprintManagement.tsx

import React from "react";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import type { TokenBlueprint } from "../../../../shell/src/shared/types/tokenBlueprint";
import { useTokenBlueprintManagement } from "../hook/useTokenBlueprintManagement";

export default function TokenBlueprintManagementPage() {
  const {
    rows,
    brandOptions,
    assigneeOptions,
    brandFilter,
    assigneeFilter,
    sortKey,
    sortDir,
    handleChangeBrandFilter,
    handleChangeAssigneeFilter,
    handleChangeSort,
    handleReset,
    handleCreate,
    handleRowClick,
  } = useTokenBlueprintManagement();

  const headers: React.ReactNode[] = [
    "トークン名",
    "シンボル",
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={handleChangeBrandFilter}
    />,
    <FilterableTableHeader
      key="assignee"
      label="担当者"
      options={assigneeOptions}
      selected={assigneeFilter}
      onChange={handleChangeAssigneeFilter}
    />,
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
    // ★ 追加：更新日列
    <SortableTableHeader
      key="updatedAt"
      label="更新日"
      sortKey="updatedAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="トークン設計"
        headerCells={headers}
        showCreateButton
        createLabel="トークン設計を作成"
        showResetButton
        onCreate={handleCreate}
        onReset={handleReset}
      >
        {rows.map((t: TokenBlueprint) => (
          <tr
            key={t.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer hover:bg-slate-50 transition-colors"
            onClick={() => handleRowClick(t.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleRowClick(t.id);
              }
            }}
          >
            <td>{t.name}</td>
            <td>{t.symbol}</td>

            {/* ★ brandName があれば brandName、無ければ brandId を文字列で表示（pill廃止） */}
            <td>{t.brandName || t.brandId}</td>

            {/* ★ 担当者は assigneeName を表示（fallback は空文字） */}
            <td>{t.assigneeName || ""}</td>

            {/* 作成日 */}
            <td>{t.createdAt}</td>

            {/* ★ 追加：更新日（yyyy/MM/dd にフォーマット済み） */}
            <td>{t.updatedAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
