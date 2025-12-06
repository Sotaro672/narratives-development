// frontend/console/inventory/src/application/inventoryManagementService.tsx

import React from "react";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import type { InventorySortKey as SortKey } from "../presentation/hook/useInventoryManagement";

/** ヘッダー生成時に必要なコンテキスト型 */
export type InventoryHeaderContext = {
  productFilter: string[];
  brandFilter: string[];
  assigneeFilter: string[];
  setProductFilter: (v: string[]) => void;
  setBrandFilter: (v: string[]) => void;
  setAssigneeFilter: (v: string[]) => void;
  sortKey: SortKey;
  sortDir: "asc" | "desc" | null;
  setSortKey: (k: SortKey) => void;
  setSortDir: (d: "asc" | "desc" | null) => void;
};

/**
 * 在庫管理一覧テーブルのヘッダー生成ロジック
 * （presentation/pages から切り出し）
 */
export function buildInventoryHeaders(
  productOptions: Array<{ value: string; label: string }>,
  brandOptions: Array<{ value: string; label: string }>,
  assigneeOptions: Array<{ value: string; label: string }>,
  ctx: InventoryHeaderContext,
): React.ReactNode[] {
  return [
    <FilterableTableHeader
      key="product"
      label="プロダクト"
      options={productOptions}
      selected={ctx.productFilter}
      onChange={(vals: string[]) => ctx.setProductFilter(vals)}
    />,
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={ctx.brandFilter}
      onChange={(vals: string[]) => ctx.setBrandFilter(vals)}
    />,
    <FilterableTableHeader
      key="assignee"
      label="担当者"
      options={assigneeOptions}
      selected={ctx.assigneeFilter}
      onChange={(vals: string[]) => ctx.setAssigneeFilter(vals)}
    />,
    <SortableTableHeader
      key="totalQuantity"
      label="総在庫数"
      sortKey="totalQuantity"
      activeKey={ctx.sortKey}
      direction={ctx.sortDir ?? null}
      onChange={(key, dir) => {
        ctx.setSortKey(key as SortKey);
        ctx.setSortDir(dir);
      }}
    />,
  ];
}
