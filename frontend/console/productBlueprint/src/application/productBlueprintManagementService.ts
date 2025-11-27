// frontend/console/productBlueprint/src/application/productBlueprintManagementService.ts

import {
  fetchProductBlueprintManagementRows as fetchRowsInfra,
  type ProductBlueprintManagementRow,
} from "../infrastructure/query/productBlueprintQuery";

export type UiRow = ProductBlueprintManagementRow;

// 一覧で使うソートキー
export type ProductBlueprintSortKey = "createdAt" | "updatedAt" | null;
export type SortDirection = "asc" | "desc" | null;

/**
 * backend から商品設計一覧を取得し、
 * Query 層の結果をそのまま UiRow[] として返すラッパー。
 * （アプリケーション層からは従来どおりこの関数を呼べばよい）
 */
export async function fetchProductBlueprintManagementRows(): Promise<UiRow[]> {
  return await fetchRowsInfra();
}

// "YYYY/MM/DD" → timestamp（ソート用）
const toTs = (yyyyMd: string) => {
  if (!yyyyMd) return 0;
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

/**
 * フィルタ & ソートの純粋関数
 */
export function filterAndSortProductBlueprintRows(params: {
  allRows: UiRow[];
  brandFilter: string[];
  assigneeFilter: string[];
  tagFilter: string[];
  sortedKey: ProductBlueprintSortKey;
  sortedDir: SortDirection;
}): UiRow[] {
  const {
    allRows,
    brandFilter,
    assigneeFilter,
    tagFilter,
    sortedKey,
    sortedDir,
  } = params;

  let work = allRows;

  // ブランド絞り込み（brandName で絞る）
  if (brandFilter.length > 0) {
    work = work.filter((r) => brandFilter.includes(r.brandName));
  }

  // 担当者絞り込み（assigneeName で絞る）
  if (assigneeFilter.length > 0) {
    work = work.filter((r) => assigneeFilter.includes(r.assigneeName));
  }

  // タグ種別絞り込み（productIdTag で絞る）
  if (tagFilter.length > 0) {
    work = work.filter((r) => tagFilter.includes(r.productIdTag));
  }

  // ソート適用（createdAt / updatedAt は "YYYY/MM/DD" 形式）
  if (sortedKey && sortedDir) {
    work = [...work].sort((a, b) => {
      const av = toTs(a[sortedKey]);
      const bv = toTs(b[sortedKey]);
      return sortedDir === "asc" ? av - bv : bv - av;
    });
  }

  return work;
}
