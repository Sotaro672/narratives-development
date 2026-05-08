// frontend\console\inventory\src\application\inventoryManagementService.tsx

import React from "react";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

import { fetchInventoryListDTO } from "../infrastructure/http/inventoryRepositoryHTTP";

// ============================================================
// Types (ViewModel for Inventory Management table)
//   columns: [productName, tokenName, availableStock, reservedCount]
//   - key: productBlueprintId + tokenBlueprintId
// ============================================================

export type InventoryManagementRow = {
  productBlueprintId: string;
  productName: string;

  tokenBlueprintId: string;
  tokenName: string;

  availableStock: number;
  reservedCount: number; // ✅ 注文数
};

export type InventorySortKey =
  | "productName"
  | "tokenName"
  | "availableStock"
  | "reservedCount";

/** ヘッダー生成時に必要なコンテキスト型 */
export type InventoryHeaderContext = {
  productFilter: string[];
  tokenFilter: string[];

  setProductFilter: (v: string[]) => void;
  setTokenFilter: (v: string[]) => void;

  sortKey: InventorySortKey;
  sortDir: "asc" | "desc" | null;
  setSortKey: (k: InventorySortKey) => void;
  setSortDir: (d: "asc" | "desc" | null) => void;
};

// ============================================================
// helpers
// ============================================================

function asString(v: unknown): string {
  return String(v ?? "").trim();
}

function asNumber(v: unknown): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

export function buildInventoryFilterOptionsFromRows(rows: InventoryManagementRow[]): {
  productOptions: Array<{ value: string; label: string }>;
  tokenOptions: Array<{ value: string; label: string }>;
} {
  const productMap = new Map<string, string>();
  const tokenMap = new Map<string, string>();

  for (const r of rows) {
    const p = asString(r.productName);
    const t = asString(r.tokenName);
    if (p) productMap.set(p, p);
    if (t) tokenMap.set(t, t);
  }

  const toOptions = (m: Map<string, string>) =>
    Array.from(m.entries()).map(([value, label]) => ({ value, label }));

  return {
    productOptions: toOptions(productMap),
    tokenOptions: toOptions(tokenMap),
  };
}

// ============================================================
// Inventory List load
// ============================================================

/**
 * ✅ 一覧DTO担当:
 * 方針:
 * - GET /inventory（ListByCurrentCompany の一覧DTO）を 1 回だけ取得
 * - inventories を (pbId, tbId) で集約して表示用 rows を返す
 */
export async function loadInventoryRowsFromBackend(): Promise<InventoryManagementRow[]> {
  console.log("[inventoryMgmt/loadInventoryRowsFromBackend] start");

  // inventories（一覧DTO）を 1 回だけ
  // 期待: [{ productBlueprintId, productName, tokenBlueprintId, tokenName, modelNumber, availableStock, reservedCount }, ...]
  const items: any[] = await fetchInventoryListDTO();

  console.log("[inventoryMgmt/loadInventoryRowsFromBackend] inventory list raw", {
    count: items.length,
    sample: items.slice(0, 5),
  });

  // (pbId, tbId) で集約
  const agg = new Map<
    string,
    {
      productBlueprintId: string;
      productName: string;
      tokenBlueprintId: string;
      tokenName: string;
      availableStock: number;
      reservedCount: number;
    }
  >();

  for (const it of items) {
    const pbId = asString(it?.productBlueprintId);
    const tbId = asString(it?.tokenBlueprintId);

    // ✅ pbId/tbId は必須（無いなら集計不能）
    if (!pbId || !tbId) {
      console.warn("[inventoryMgmt/loadInventoryRowsFromBackend] skip: missing ids", {
        pbId,
        tbId,
        raw: it,
      });
      continue;
    }

    const productName = asString(it?.productName) || "-";
    const tokenName = asString(it?.tokenName) || tbId;

    // ✅ 在庫数(表示)は availableStock を正（無ければ stock）
    const availableStock = asNumber(it?.availableStock ?? it?.stock);

    // ✅ 注文数（reservedCount）
    const reservedCount = asNumber(it?.reservedCount);

    const key = `${pbId}__${tbId}`;
    const cur = agg.get(key);
    if (!cur) {
      agg.set(key, {
        productBlueprintId: pbId,
        productName,
        tokenBlueprintId: tbId,
        tokenName,
        availableStock,
        reservedCount,
      });
    } else {
      cur.availableStock += availableStock;
      cur.reservedCount += reservedCount;
      // productName / tokenName は先勝ち
    }
  }

  const out: InventoryManagementRow[] = Array.from(agg.values()).map((v) => ({
    productBlueprintId: v.productBlueprintId,
    productName: v.productName || "-",
    tokenBlueprintId: v.tokenBlueprintId,
    tokenName: v.tokenName || "-",
    availableStock: v.availableStock,
    reservedCount: v.reservedCount,
  }));

  console.log("[inventoryMgmt/loadInventoryRowsFromBackend] done", {
    rows: out.length,
    sample: out.slice(0, 10),
  });

  return out;
}

// ============================================================
// UI header builder
// ============================================================

/**
 * 在庫管理一覧テーブルのヘッダー生成ロジック
 * 列順: [プロダクト名, トークン名, 在庫数, 注文数]
 */
export function buildInventoryHeaders(
  productOptions: Array<{ value: string; label: string }>,
  tokenOptions: Array<{ value: string; label: string }>,
  ctx: InventoryHeaderContext,
): React.ReactNode[] {
  return [
    <FilterableTableHeader
      key="productName"
      label="プロダクト名"
      options={productOptions}
      selected={ctx.productFilter}
      onChange={(vals: string[]) => ctx.setProductFilter(vals)}
    />,
    <FilterableTableHeader
      key="tokenName"
      label="トークン名"
      options={tokenOptions}
      selected={ctx.tokenFilter}
      onChange={(vals: string[]) => ctx.setTokenFilter(vals)}
    />,
    <SortableTableHeader
      key="availableStock"
      label="在庫数"
      sortKey="availableStock"
      activeKey={ctx.sortKey}
      direction={ctx.sortDir ?? null}
      onChange={(key, dir) => {
        ctx.setSortKey(key as InventorySortKey);
        ctx.setSortDir(dir);
      }}
    />,
    <SortableTableHeader
      key="reservedCount"
      label="注文数"
      sortKey="reservedCount"
      activeKey={ctx.sortKey}
      direction={ctx.sortDir ?? null}
      onChange={(key, dir) => {
        ctx.setSortKey(key as InventorySortKey);
        ctx.setSortDir(dir);
      }}
    />,
  ];
}