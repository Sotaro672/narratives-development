// frontend/console/productBlueprint/src/application/productBlueprintDeletedService.ts

import { listDeletedProductBlueprintsHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";
import { fetchBrandNameById } from "../../../brand/src/infrastructure/http/brandRepositoryHTTP";
import { fetchMemberDisplayNameById } from "../../../member/src/infrastructure/http/memberRepositoryHTTP";

/**
 * 一覧画面用の UI 行型
 * - brandName / assigneeName は ID から解決した表示名
 * - deletedAt / expireAt は "YYYY/MM/DD" 形式の文字列
 */
export type DeletedUiRow = {
  id: string;
  productName: string;
  brandName: string;
  assigneeName: string;
  deletedAt: string; // YYYY/MM/DD
  expireAt: string; // YYYY/MM/DD
};

// backend /product-blueprints/deleted (想定) のレスポンス行
type RawDeletedProductBlueprintRow = {
  id?: string;
  productName?: string;

  brandId?: string;
  assigneeId?: string;

  deletedAt?: string | null;
  expireAt?: string | null;
};

export type ProductBlueprintDeletedSortKey = "deletedAt" | "expireAt" | null;
export type SortDirection = "asc" | "desc" | null;

// "YYYY/MM/DD" → timestamp（ソート用）
const toTs = (yyyyMd: string) => {
  if (!yyyyMd) return 0;
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

// ISO / RFC 系 or 任意文字列 → "YYYY/MM/DD"（壊れてたらそのまま返す）
const toDisplayDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso ?? "";
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
};

/**
 * backend から論理削除済みの商品設計一覧を取得し、
 * - brandId → brandName 変換
 * - assigneeId → assigneeName 変換
 * を行って DeletedUiRow[] を構築する。
 */
export async function fetchProductBlueprintDeletedRows(): Promise<DeletedUiRow[]> {
  const list = await listDeletedProductBlueprintsHTTP();

  const uiRows: DeletedUiRow[] = [];

  for (const pb of list as RawDeletedProductBlueprintRow[]) {
    // 念のため deletedAt が無い行はスキップ
    if (!pb.deletedAt) continue;

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

    const deletedAtDisp = toDisplayDate(pb.deletedAt ?? "");
    const expireAtDisp = toDisplayDate(pb.expireAt ?? "");

    uiRows.push({
      id: pb.id ?? "",
      productName: pb.productName ?? "",
      brandName,
      assigneeName,
      deletedAt: deletedAtDisp,
      expireAt: expireAtDisp,
    });
  }

  return uiRows;
}

/**
 * フィルタ & ソートの純粋関数
 * - brandFilter: brandName で絞り込み
 * - assigneeFilter: assigneeName で絞り込み
 * - sortedKey: "deletedAt" / "expireAt" でソート
 */
export function filterAndSortProductBlueprintDeletedRows(params: {
  allRows: DeletedUiRow[];
  brandFilter: string[];
  assigneeFilter: string[];
  sortedKey: ProductBlueprintDeletedSortKey;
  sortedDir: SortDirection;
}): DeletedUiRow[] {
  const { allRows, brandFilter, assigneeFilter, sortedKey, sortedDir } = params;

  let work = allRows;

  // ブランド絞り込み（brandName で絞る）
  if (brandFilter.length > 0) {
    work = work.filter((r) => brandFilter.includes(r.brandName));
  }

  // 担当者絞り込み（assigneeName で絞る）
  if (assigneeFilter.length > 0) {
    work = work.filter((r) => assigneeFilter.includes(r.assigneeName));
  }

  // ソート適用（deletedAt / expireAt は "YYYY/MM/DD" 形式）
  if (sortedKey && sortedDir) {
    work = [...work].sort((a, b) => {
      const av = toTs(a[sortedKey]);
      const bv = toTs(b[sortedKey]);
      return sortedDir === "asc" ? av - bv : bv - av;
    });
  }

  return work;
}
