// frontend/console/productBlueprint/src/infrastructure/query/productBlueprintQuery.ts

import { listProductBlueprintsHTTP } from "../repository/productBlueprintRepositoryHTTP";
import { fetchBrandNameById } from "../../../../brand/src/infrastructure/http/brandRepositoryHTTP";
import { fetchMemberDisplayNameById } from "../../../../member/src/infrastructure/http/memberRepositoryHTTP";

export type ProductBlueprintManagementRow = {
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
  // deletedAt はバックエンド側でフィルタされるため、ここでは参照しない
};

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
 * backend から商品設計一覧を取得し、
 * - brandId → brandName 変換
 * - assigneeId → assigneeName 変換
 * を行って ProductBlueprintManagementRow[] を構築する。
 *
 * ※ 論理削除済みの除外は backend (Usecase.List) 側で実施済み。
 */
export async function fetchProductBlueprintManagementRows(): Promise<ProductBlueprintManagementRow[]> {
  const list = await listProductBlueprintsHTTP();

  const uiRows: ProductBlueprintManagementRow[] = [];

  for (const pb of list as RawProductBlueprintListRow[]) {
    // 🚫 deletedAt によるフィルタリングは backend 側で実施済み

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
    const productIdTag = (pb.productIdTag ?? "").trim() || "-";

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
