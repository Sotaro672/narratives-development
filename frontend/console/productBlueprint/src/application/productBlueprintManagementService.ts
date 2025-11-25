// frontend/console/productBlueprint/src/application/productBlueprintManagementService.ts

import { listProductBlueprintsHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";
import { fetchBrandNameById } from "../../../brand/src/infrastructure/http/brandRepositoryHTTP";
import { fetchMemberDisplayNameById } from "../../../member/src/infrastructure/http/memberRepositoryHTTP";

export type UiRow = {
  id: string;
  productName: string;
  brandName: string;
  assigneeName: string;
  productIdTag: string;
  createdAt: string; // YYYY/MM/DD
  updatedAt: string; // YYYY/MM/DD
};

// backend /product-blueprints のレスポンス想定
type RawProductBlueprintListRow = {
  id?: string;
  productName?: string;

  brandId?: string;
  assigneeId?: string;

  // backend の JSON は "productIdTag": "QRコード" などの文字列を直接返す想定
  productIdTag?: string | null;

  createdAt?: string; // "YYYY/MM/DD" を想定（handler でフォーマット済み）
  updatedAt?: string; // "YYYY/MM/DD"
  deletedAt?: string | null;
};

export type ProductBlueprintSortKey = "createdAt" | "updatedAt" | null;
export type SortDirection = "asc" | "desc" | null;

// ISO っぽいもの / YYYY/MM/DD → "YYYY/MM/DD"（壊れてたらそのまま返す）
const toDisplayDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso ?? "";
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
};

// "YYYY/MM/DD" → timestamp（ソート用）
const toTs = (yyyyMd: string) => {
  if (!yyyyMd) return 0;
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

/**
 * backend から商品設計一覧を取得し、
 * - brandId → brandName 変換
 * - assigneeId → assigneeName 変換
 * を行って UiRow[] を構築する。
 */
export async function fetchProductBlueprintManagementRows(): Promise<UiRow[]> {
  const list = await listProductBlueprintsHTTP();

  const uiRows: UiRow[] = [];

  for (const pb of list as RawProductBlueprintListRow[]) {
    if (pb.deletedAt) continue;

    // ブランド名変換
    const brandId = pb.brandId ?? "";
    const brandName = brandId ? await fetchBrandNameById(brandId) : "";

    // 担当者名変換 (assigneeId -> displayName)
    const assigneeId = (pb.assigneeId ?? "").trim();
    let assigneeName = "-";
    if (assigneeId) {
      const displayName = await fetchMemberDisplayNameById(assigneeId);
      assigneeName = displayName.trim() || assigneeId;
    }

    // ProductIDTag（そのまま表示。空なら "-"）
    const productIdTagRaw = (pb.productIdTag ?? "").trim();
    const productIdTag = productIdTagRaw || "-";

    // 日付整形
    const createdAtDisp = toDisplayDate(pb.createdAt ?? "");
    const updatedAtDisp = toDisplayDate(pb.updatedAt ?? pb.createdAt ?? "");

    uiRows.push({
      id: pb.id ?? "",
      productName: pb.productName ?? "",
      brandName,
      assigneeName,
      productIdTag,
      createdAt: createdAtDisp,
      updatedAt: updatedAtDisp,
    });
  }

  return uiRows;
}

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
