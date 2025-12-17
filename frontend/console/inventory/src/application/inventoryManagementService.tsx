// frontend/console/inventory/src/application/inventoryManagementService.tsx

import React from "react";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

// Firebase Auth から ID トークンを取得
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

import { fetchPrintedInventorySummaries } from "../infrastructure/http/inventoryRepositoryHTTP";
import type { InventoryProductSummary } from "../infrastructure/http/inventoryRepositoryHTTP";

// ============================================================
// Backend base URL
// ============================================================

const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("Not authenticated");
  }
  const token = await user.getIdToken();
  if (!token) {
    throw new Error("Failed to acquire ID token");
  }
  return token;
}

// ============================================================
// Inventory Query DTO (GET /inventory?productBlueprintId=...)
// ============================================================

type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  assigneeId?: string | null;
};

type ProductBlueprintSummaryDTO = {
  id: string;
  name?: string;
};

type InventoryDetailRowDTO = {
  tokenBlueprintId?: string; // ★追加: 集計キーとして使う
  token?: string;
  modelNumber: string;
  stock: number;
  // size/color/rgb などは一覧表示では使わない
};

type InventoryDetailDTO = {
  inventoryId: string; // pbId が入る想定（互換）
  productBlueprintId: string;
  productBlueprintPatch?: ProductBlueprintPatchDTO;
  productBlueprint?: ProductBlueprintSummaryDTO;
  rows: InventoryDetailRowDTO[];
  totalStock: number;
  updatedAt?: string;
};

async function fetchInventoryDetailByProductBlueprintId(
  productBlueprintId: string,
): Promise<InventoryDetailDTO> {
  const token = await getIdTokenOrThrow();

  const url = `${API_BASE}/inventory?productBlueprintId=${encodeURIComponent(
    productBlueprintId,
  )}`;

  console.log("[inventoryMgmt/fetchInventoryDetailByPB] start", {
    productBlueprintId,
    url,
  });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.warn("[inventoryMgmt/fetchInventoryDetailByPB] failed", {
      productBlueprintId,
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(
      `Failed to fetch inventory by productBlueprintId: ${res.status} ${res.statusText}`,
    );
  }

  const data = (await res.json()) as any;

  const mapped: InventoryDetailDTO = {
    inventoryId: String(data?.inventoryId ?? ""),
    productBlueprintId: String(data?.productBlueprintId ?? productBlueprintId ?? ""),
    productBlueprintPatch: data?.productBlueprintPatch ?? undefined,
    productBlueprint: data?.productBlueprint
      ? {
          id: String(data.productBlueprint.id ?? ""),
          name: data.productBlueprint.name
            ? String(data.productBlueprint.name)
            : undefined,
        }
      : undefined,
    rows: Array.isArray(data?.rows)
      ? data.rows.map((r: any) => ({
          tokenBlueprintId: r?.tokenBlueprintId
            ? String(r.tokenBlueprintId)
            : undefined, // ★追加
          token: r?.token ? String(r.token) : undefined,
          modelNumber: String(r?.modelNumber ?? ""),
          stock: Number(r?.stock ?? 0),
        }))
      : [],
    totalStock: Number(data?.totalStock ?? 0),
    updatedAt: data?.updatedAt ? String(data.updatedAt) : undefined,
  };

  console.log("[inventoryMgmt/fetchInventoryDetailByPB] ok", {
    productBlueprintId,
    rowsCount: mapped.rows.length,
    totalStock: mapped.totalStock,
    sampleRows: mapped.rows.slice(0, 5),
  });

  return mapped;
}

// ============================================================
// ViewModel for Inventory Management table
//   columns: [productName, tokenName, stock]
//   - modelNumber 列は廃止（期待値: pbId + tokenBlueprintId で1行集約）
// ============================================================

export type InventoryManagementRow = {
  productBlueprintId: string;
  productName: string;

  tokenBlueprintId: string; // ★追加: 集計キー
  tokenName: string;

  stock: number;
};

export type InventorySortKey = "productName" | "tokenName" | "stock";

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

export function buildInventoryFilterOptionsFromRows(rows: InventoryManagementRow[]): {
  productOptions: Array<{ value: string; label: string }>;
  tokenOptions: Array<{ value: string; label: string }>;
} {
  const productMap = new Map<string, string>();
  const tokenMap = new Map<string, string>();

  for (const r of rows) {
    const p = String(r.productName ?? "").trim();
    const t = String(r.tokenName ?? "").trim();
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

/**
 * ✅ inventory_query.go の結果を画面表示用にロードする
 *
 * 方針:
 * 1) printed の ProductBlueprint 一覧を取得（既存の入口）
 * 2) 各 productBlueprintId について GET /inventory?productBlueprintId=... を叩く
 * 3) rows を [tokenBlueprintId] で集計して一覧用行にする（型番行は作らない）
 */
export async function loadInventoryRowsFromBackend(): Promise<InventoryManagementRow[]> {
  console.log("[inventoryMgmt/loadInventoryRowsFromBackend] start");

  // ① printed product blueprints（一覧の母集団）
  const summaries: InventoryProductSummary[] = await fetchPrintedInventorySummaries();

  console.log("[inventoryMgmt/loadInventoryRowsFromBackend] printed summaries", {
    count: summaries.length,
    sample: summaries.slice(0, 5),
  });

  const out: InventoryManagementRow[] = [];

  // ② pbId ごとに inventory query を叩く（並列）
  const tasks = summaries.map(async (s) => {
    const pbId = String(s.id ?? "").trim();
    if (!pbId) return;

    try {
      const dto = await fetchInventoryDetailByProductBlueprintId(pbId);

      // productName: patch -> summary -> fallback
      const productName =
        String(dto?.productBlueprint?.name ?? "").trim() ||
        String(dto?.productBlueprintPatch?.productName ?? "").trim() ||
        String(s.productName ?? "").trim() ||
        "-";

      // ③ rows を tokenBlueprintId で集計（表示名 token は保持、modelNumber は無視）
      const agg = new Map<
        string,
        { tokenBlueprintId: string; tokenName: string; stock: number }
      >();

      const rows = Array.isArray(dto.rows) ? dto.rows : [];
      for (const r of rows) {
        const tokenBlueprintId = String(r?.tokenBlueprintId ?? "").trim();
        if (!tokenBlueprintId) {
          // backend が未対応 or データ欠損の場合は「集計できない」のでスキップ（ログだけ出す）
          console.warn("[inventoryMgmt/loadInventoryRowsFromBackend] row missing tokenBlueprintId", {
            productBlueprintId: pbId,
            row: r,
          });
          continue;
        }

        const tokenName = String(r?.token ?? "").trim() || "-";
        const stock = Number(r?.stock ?? 0);

        const key = tokenBlueprintId;
        const cur = agg.get(key);
        if (!cur) {
          agg.set(key, { tokenBlueprintId, tokenName, stock });
        } else {
          cur.stock += stock;
          // tokenName は先勝ち（後続で変わっても一覧は壊さない）
        }
      }

      // rows が空（or 全行 tokenBlueprintId 欠損）なら totalStock を 1行で出す（最低限の見え方）
      if (agg.size === 0) {
        const fallbackStock = Number(dto.totalStock ?? 0);
        if (fallbackStock > 0) {
          out.push({
            productBlueprintId: pbId,
            productName,
            tokenBlueprintId: "-", // 不明
            tokenName: "-",
            stock: fallbackStock,
          });
        }
        return;
      }

      for (const v of agg.values()) {
        out.push({
          productBlueprintId: pbId,
          productName,
          tokenBlueprintId: v.tokenBlueprintId,
          tokenName: v.tokenName,
          stock: v.stock,
        });
      }
    } catch (e: any) {
      // inventory が無い pb は一覧から落として OK（必要ならここで 0 行を作る）
      console.warn("[inventoryMgmt/loadInventoryRowsFromBackend] skip pbId (fetch failed)", {
        productBlueprintId: pbId,
        error: String(e?.message ?? e),
      });
    }
  });

  await Promise.all(tasks);

  console.log("[inventoryMgmt/loadInventoryRowsFromBackend] done", {
    rows: out.length,
    sample: out.slice(0, 10),
  });

  return out;
}

/**
 * 在庫管理一覧テーブルのヘッダー生成ロジック
 * 列順: [プロダクト名, トークン名, 在庫数]
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
  ];
}
