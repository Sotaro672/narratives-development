import React from "react";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

import {
  fetchPrintedInventorySummaries,
  fetchInventoryListDTO,
  type InventoryProductSummary,
} from "../infrastructure/http/inventoryRepositoryHTTP";

// ============================================================
// Types (ViewModel for Inventory Management table)
//   columns: [productName, tokenName, stock, reservedCount]
//   - key: productBlueprintId + tokenBlueprintId
// ============================================================

export type InventoryManagementRow = {
  productBlueprintId: string;
  productName: string;

  tokenBlueprintId: string;
  tokenName: string;

  stock: number; // (= availableStock)
  reservedCount: number; // ✅ 注文数
};

export type InventorySortKey = "productName" | "tokenName" | "stock" | "reservedCount";

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
// 方針A（確定版）:
// - backend /inventory は tokenBlueprintId を返す前提
// - 文字揺れ吸収や tbId="-" の暫定は削除
// ============================================================

/**
 * ✅ 一覧DTO担当:
 * 方針:
 * 1) printed の ProductBlueprint 一覧を取得（一覧の母集団）
 * 2) GET /inventory（ListByCurrentCompany の一覧DTO）を 1 回だけ取得
 * 3) inventories を (pbId, tbId) で集約して表示用 rows を返す
 */
export async function loadInventoryRowsFromBackend(): Promise<InventoryManagementRow[]> {
  console.log("[inventoryMgmt/loadInventoryRowsFromBackend] start");

  // ① printed product blueprints（一覧の母集団）
  const summaries: InventoryProductSummary[] = await fetchPrintedInventorySummaries();

  console.log("[inventoryMgmt/loadInventoryRowsFromBackend] printed summaries", {
    count: summaries.length,
    sample: summaries.slice(0, 5),
  });

  const printedByPbId = new Map<string, InventoryProductSummary>();
  for (const s of summaries) {
    const pbId = asString(s.id);
    if (pbId) printedByPbId.set(pbId, s);
  }

  // ② inventories（一覧DTO）を 1 回だけ
  // 期待: [{ productBlueprintId, productName, tokenBlueprintId, tokenName, modelNumber, stock, availableStock, reservedCount }, ...]
  const items: any[] = await fetchInventoryListDTO();

  console.log("[inventoryMgmt/loadInventoryRowsFromBackend] inventory list raw", {
    count: items.length,
    sample: items.slice(0, 5),
  });

  // ③ (pbId, tbId) で集約
  const agg = new Map<
    string,
    {
      productBlueprintId: string;
      tokenBlueprintId: string;
      tokenName: string;
      stock: number; // (= availableStock)
      reservedCount: number;
    }
  >();

  for (const it of items) {
    const pbId = asString(it?.productBlueprintId);
    const tbId = asString(it?.tokenBlueprintId);

    // ✅ 方針A: pbId/tbId は必須（無いなら集計不能）
    if (!pbId || !tbId) {
      console.warn("[inventoryMgmt/loadInventoryRowsFromBackend] skip: missing ids", {
        pbId,
        tbId,
        raw: it,
      });
      continue;
    }

    // printed に含まれない PB の在庫は一覧では出さない
    if (!printedByPbId.has(pbId)) continue;

    const tokenName = asString(it?.tokenName) || tbId;

    // ✅ stock は availableStock 優先（無ければ stock）
    const stock = asNumber(it?.availableStock ?? it?.stock);

    // ✅ 注文数（reservedCount）
    const reservedCount = asNumber(it?.reservedCount);

    // ✅ 0行(在庫0/注文0)も除外しない（inventory テーブルがある限り list したい要件）
    // if (stock <= 0 && reservedCount <= 0) continue;

    const key = `${pbId}__${tbId}`;
    const cur = agg.get(key);
    if (!cur) {
      agg.set(key, {
        productBlueprintId: pbId,
        tokenBlueprintId: tbId,
        tokenName,
        stock,
        reservedCount,
      });
    } else {
      cur.stock += stock;
      cur.reservedCount += reservedCount;
      // tokenName は先勝ち
    }
  }

  const out: InventoryManagementRow[] = [];
  for (const v of agg.values()) {
    const s = printedByPbId.get(v.productBlueprintId);
    const productName = asString(s?.productName) || "-";

    out.push({
      productBlueprintId: v.productBlueprintId,
      productName,
      tokenBlueprintId: v.tokenBlueprintId, // ✅ "-" 埋めをしない
      tokenName: v.tokenName || "-",
      stock: v.stock,
      reservedCount: v.reservedCount,
    });
  }

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
      key="stock"
      label="在庫数"
      sortKey="stock"
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
