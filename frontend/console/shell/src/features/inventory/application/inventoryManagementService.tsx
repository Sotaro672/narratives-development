// frontend/console/shell/src/features/inventory/application/inventoryManagementService.tsx

import React from "react";

import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../layout/List/List";

import { getInventoryListRaw } from "../infrastructure/inventoryApi";

// ============================================================
// Types（Inventory Management table ViewModel）
//
// columns:
// - productName
// - tokenName
// - shippingAddressName
// - availableStock
// - reservedCount
//
// key:
// - productBlueprintId + tokenBlueprintId
// ============================================================

export type InventoryManagementRow = {
  productBlueprintId: string;
  productName: string;
  tokenBlueprintId: string;
  tokenName: string;
  shippingAddressName: string;
  availableStock: number;
  reservedCount: number;
};

export type InventorySortKey =
  | "productName"
  | "tokenName"
  | "availableStock"
  | "reservedCount";

/**
 * ヘッダー生成時に必要なコンテキスト。
 */
export type InventoryHeaderContext = {
  productFilter: string[];
  tokenFilter: string[];
  setProductFilter: (values: string[]) => void;
  setTokenFilter: (values: string[]) => void;
  sortKey: InventorySortKey;
  sortDir: "asc" | "desc" | null;
  setSortKey: (key: InventorySortKey) => void;
  setSortDir: (direction: "asc" | "desc" | null) => void;
};

// ============================================================
// Filter options
// ============================================================

export function buildInventoryFilterOptionsFromRows(
  rows: InventoryManagementRow[],
): {
  productOptions: Array<{ value: string; label: string }>;
  tokenOptions: Array<{ value: string; label: string }>;
} {
  const productMap = new Map<string, string>();
  const tokenMap = new Map<string, string>();

  for (const row of rows) {
    if (row.productName) {
      productMap.set(row.productName, row.productName);
    }

    if (row.tokenName) {
      tokenMap.set(row.tokenName, row.tokenName);
    }
  }

  const toOptions = (
    source: Map<string, string>,
  ): Array<{ value: string; label: string }> => {
    return Array.from(source.entries()).map(([value, label]) => ({
      value,
      label,
    }));
  };

  return {
    productOptions: toOptions(productMap),
    tokenOptions: toOptions(tokenMap),
  };
}

// ============================================================
// Inventory List load
// ============================================================

/**
 * Inventory一覧を取得し、画面表示単位に集約する。
 *
 * 方針:
 * - GET /inventoryを1回だけ呼び出す
 * - getInventoryListRawの戻り値を正とする
 * - productBlueprintIdとtokenBlueprintIdの組み合わせで集約する
 * - shippingAddressNameはbackendでshippingAddressIdから解決済みの値を使用する
 *
 * 画面では同一ProductBlueprint・TokenBlueprintの在庫数と
 * 注文数を合算する。
 */
export async function loadInventoryRowsFromBackend(): Promise<
  InventoryManagementRow[]
> {
  const items = await getInventoryListRaw();

  const aggregatedRows = new Map<string, InventoryManagementRow>();

  for (const item of items) {
    const productBlueprintId = item.productBlueprintId;
    const tokenBlueprintId = item.tokenBlueprintId;

    if (!productBlueprintId || !tokenBlueprintId) {
      continue;
    }

    const key = `${productBlueprintId}__${tokenBlueprintId}`;
    const current = aggregatedRows.get(key);

    if (!current) {
      aggregatedRows.set(key, {
        productBlueprintId,
        productName: item.productName || "-",
        tokenBlueprintId,
        tokenName: item.tokenName || tokenBlueprintId,
        shippingAddressName: item.shippingAddressName || "",
        availableStock: item.availableStock,
        reservedCount: item.reservedCount,
      });
      continue;
    }

    if (!current.shippingAddressName && item.shippingAddressName) {
      current.shippingAddressName = item.shippingAddressName;
    }

    current.availableStock += item.availableStock;
    current.reservedCount += item.reservedCount;
  }

  return Array.from(aggregatedRows.values());
}

// ============================================================
// UI header builder
// ============================================================

/**
 * 在庫管理一覧テーブルのヘッダーを生成する。
 *
 * 列順:
 * - プロダクト名
 * - トークン名
 * - 保管場所
 * - 在庫数
 * - 注文数
 */
export function buildInventoryHeaders(
  productOptions: Array<{ value: string; label: string }>,
  tokenOptions: Array<{ value: string; label: string }>,
  context: InventoryHeaderContext,
): React.ReactNode[] {
  return [
    <FilterableTableHeader
      key="productName"
      label="プロダクト名"
      options={productOptions}
      selected={context.productFilter}
      onChange={(values: string[]) => {
        context.setProductFilter(values);
      }}
    />,

    <FilterableTableHeader
      key="tokenName"
      label="トークン名"
      options={tokenOptions}
      selected={context.tokenFilter}
      onChange={(values: string[]) => {
        context.setTokenFilter(values);
      }}
    />,

    <span key="shippingAddressName">保管場所</span>,

    <SortableTableHeader
      key="availableStock"
      label="在庫数"
      sortKey="availableStock"
      activeKey={context.sortKey}
      direction={context.sortDir ?? null}
      onChange={(key, direction) => {
        context.setSortKey(key as InventorySortKey);
        context.setSortDir(direction);
      }}
    />,

    <SortableTableHeader
      key="reservedCount"
      label="注文数"
      sortKey="reservedCount"
      activeKey={context.sortKey}
      direction={context.sortDir ?? null}
      onChange={(key, direction) => {
        context.setSortKey(key as InventorySortKey);
        context.setSortDir(direction);
      }}
    />,
  ];
}