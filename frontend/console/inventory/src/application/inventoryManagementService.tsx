// frontend/console/inventory/src/application/inventoryManagementService.tsx

import React from "react";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import type { InventorySortKey as SortKey } from "../presentation/hook/useInventoryManagement";
import type { InventoryProductSummary } from "../infrastructure/http/inventoryRepositoryHTTP";
import { fetchPrintedInventorySummaries } from "../infrastructure/http/inventoryRepositoryHTTP";

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
 * inventoryRepositoryHTTP.ts から取得した
 * productName / brandId / assigneeId の配列を
 * フィルター用 options に変換するヘルパー
 */
export function buildInventoryFilterOptionsFromSummaries(
  summaries: InventoryProductSummary[],
): {
  productOptions: Array<{ value: string; label: string }>;
  brandOptions: Array<{ value: string; label: string }>;
  assigneeOptions: Array<{ value: string; label: string }>;
} {
  const productMap = new Map<string, string>();
  const brandMap = new Map<string, string>();
  const assigneeMap = new Map<string, string>();

  for (const s of summaries) {
    // プロダクト: value=id / label=productName
    if (s.id && s.productName) {
      productMap.set(s.id, s.productName);
    }
    // ブランド・担当者は現状 ID ベース（名前解決は別サービスで）
    if (s.brandId) {
      brandMap.set(s.brandId, s.brandId);
    }
    if (s.assigneeId) {
      assigneeMap.set(s.assigneeId, s.assigneeId);
    }
  }

  const toOptions = (m: Map<string, string>) =>
    Array.from(m.entries()).map(([value, label]) => ({ value, label }));

  return {
    productOptions: toOptions(productMap),
    brandOptions: toOptions(brandMap),
    assigneeOptions: toOptions(assigneeMap),
  };
}

/**
 * 在庫管理一覧用：
 * backend の ListPrinted(ctx, ids []string) を
 * HTTP 経由で叩くエンドポイント（GET /product-blueprints/printed）
 * の結果を使って、フィルター options を構築する。
 */
export async function loadInventoryFilterOptionsFromBackend(): Promise<{
  productOptions: Array<{ value: string; label: string }>;
  brandOptions: Array<{ value: string; label: string }>;
  assigneeOptions: Array<{ value: string; label: string }>;
}> {
  // ★ ここで裏側では ProductBlueprintUsecase.ListPrinted(ctx) が呼ばれる
  const summaries = await fetchPrintedInventorySummaries();
  return buildInventoryFilterOptionsFromSummaries(summaries);
}

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
