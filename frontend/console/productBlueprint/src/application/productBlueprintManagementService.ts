// frontend/console/productBlueprint/src/application/productBlueprintManagementService.ts

import { listProductBlueprintsHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";
import { fetchBrandNameById } from "../../../brand/src/infrastructure/http/brandRepositoryHTTP";

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

  productIdTag?: {
    type?: string | null;
  } | null;

  createdAt?: string; // ISO8601
  updatedAt?: string; // ISO8601
  deletedAt?: string | null;
};

export type ProductBlueprintSortKey = "createdAt" | "updatedAt" | null;
export type SortDirection = "asc" | "desc" | null;

// ISO8601 → "YYYY/MM/DD"（壊れてたらそのまま返す） ※一覧用
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
 * backend から商品設計一覧を取得し、brandId → brandName 変換も行う
 * - brandName: backend brand.Service.GetNameByID 相当（fetchBrandNameById）
 */
export async function fetchProductBlueprintManagementRows(): Promise<UiRow[]> {
  const list = await listProductBlueprintsHTTP();

  const uiRows: UiRow[] = [];

  for (const pb of list as RawProductBlueprintListRow[]) {
    if (pb.deletedAt) continue;

    const brandId = pb.brandId ?? "";
    const brandName = await fetchBrandNameById(brandId);

    // ProductIDTag.type は "qr" | "nfc" 想定。表示上は大文字ラベル化。
    const productIdTag =
      pb.productIdTag && pb.productIdTag.type
        ? String(pb.productIdTag.type).toUpperCase()
        : "-";

    const createdAtDisp = toDisplayDate(pb.createdAt ?? "");
    const updatedAtDisp = toDisplayDate(pb.updatedAt ?? pb.createdAt ?? "");

    uiRows.push({
      id: pb.id ?? "",
      productName: pb.productName ?? "",
      brandName,
      assigneeName: pb.assigneeId ?? "-",
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
  sortedKey: ProductBlueprintSortKey;
  sortedDir: SortDirection;
}): UiRow[] {
  const { allRows, brandFilter, sortedKey, sortedDir } = params;

  let work = allRows;

  // ブランド絞り込み（brandName で絞る）
  if (brandFilter.length > 0) {
    work = work.filter((r) => brandFilter.includes(r.brandName));
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
