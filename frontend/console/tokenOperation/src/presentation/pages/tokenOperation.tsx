// frontend/tokenOperation/src/presentation/pages/tokenOperation.tsx

import React from "react";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { useTokenOperation } from "../hook/useTokenOperation";

export default function TokenOperationPage() {
  const {
    brandFilter,
    assigneeFilter,
    brandOptions,
    assigneeOptions,
    activeKey,
    direction,
    rows,
    handleSortChange,
    handleBrandFilterChange,
    handleAssigneeFilterChange,
    handleReset,
    goDetail,
  } = useTokenOperation();

  // ── Table headers（★ID列を削除） ─────────────────────────
  const headers: React.ReactNode[] = [
    <SortableTableHeader
      key="tokenName"
      label="トークン名"
      sortKey="tokenName"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        handleSortChange(key, dir);
      }}
    />,
    <SortableTableHeader
      key="symbol"
      label="シンボル"
      sortKey="symbol"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        handleSortChange(key, dir);
      }}
    />,
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={handleBrandFilterChange}
    />,
    <FilterableTableHeader
      key="assignee"
      label="担当者"
      options={assigneeOptions}
      selected={assigneeFilter}
      onChange={handleAssigneeFilterChange}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="トークン運用"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={handleReset}
      >
        {rows.map((t) => (
          <tr
            key={t.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => goDetail(t.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(t.id);
              }
            }}
          >
            {/* ★ ID 列を削除 */}
            <td>{t.tokenName}</td>
            <td>{t.symbol}</td>
            <td>
              <span className="lp-brand-pill">{t.brandName}</span>
            </td>
            <td>{t.assigneeName}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
