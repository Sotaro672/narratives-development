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
    mintedOptions,
    brandFilter,
    assigneeFilter,
    mintedFilter,
    sortKey,
    sortDir,
    handleChangeBrandFilter,
    handleChangeAssigneeFilter,
    handleChangeMintedFilter,
    handleChangeSort,
    handleReset,
    handleCreate,
    handleRowClick,
    isResetting,
  } = useTokenBlueprintManagement();

  const headers: React.ReactNode[] = [
    "トークン名",
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
    <FilterableTableHeader
      key="minted"
      label="ミント"
      options={mintedOptions}
      selected={mintedFilter}
      onChange={handleChangeMintedFilter}
    />,
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={handleChangeSort}
    />,
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
        isResetting={isResetting}
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
            <td>{t.brandName || t.brandId}</td>
            <td>{t.assigneeName || ""}</td>
            <td>{String(t.minted)}</td>
            <td>{t.createdAt}</td>
            <td>{t.updatedAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}