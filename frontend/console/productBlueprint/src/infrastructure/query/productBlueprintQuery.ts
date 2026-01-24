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

  /**
   * 修正案B: backend は ISO8601/RFC3339 の日時文字列を返す。
   * この層では表示整形を行わず、raw の日時文字列を保持して presentation に渡す。
   */
  createdAt: string; // ISO8601/RFC3339 datetime string
  updatedAt: string; // ISO8601/RFC3339 datetime string
};

// backend /product-blueprints のレスポンス想定（修正案B固定）
type RawProductBlueprintListRow = {
  id?: string;
  productName?: string;

  brandId?: string;
  assigneeId?: string;

  // backend の JSON は "productIdTag": "QRコード" などの文字列を直接返す想定
  productIdTag?: string | null;

  // backend は createdAt/updatedAt を ISO8601/RFC3339 の文字列で返す前提
  createdAt?: string | null;
  updatedAt?: string | null;

  // deletedAt はバックエンド側でフィルタされるため、ここでは参照しない
};

function s(v: unknown): string {
  return v == null ? "" : String(v).trim();
}

/**
 * backend から商品設計一覧を取得し、
 * - brandId → brandName 変換
 * - assigneeId → assigneeName 変換
 * を行って ProductBlueprintManagementRow[] を構築する。
 *
 * ※ 論理削除済みの除外は backend (Usecase.List) 側で実施済み。
 *
 * 修正案B:
 * - createdAt/updatedAt の「表示整形」はこの層で行わない（時刻情報を保持）
 */
export async function fetchProductBlueprintManagementRows(): Promise<ProductBlueprintManagementRow[]> {
  const list = await listProductBlueprintsHTTP();

  const uiRows: ProductBlueprintManagementRow[] = [];

  for (const pb of list as RawProductBlueprintListRow[]) {
    // ブランド名変換
    const brandId = s(pb.brandId);
    const brandName = brandId ? await fetchBrandNameById(brandId) : "";

    // 担当者名変換 (assigneeId -> displayName)
    const assigneeId = s(pb.assigneeId);
    let assigneeName = "-";
    if (assigneeId) {
      const displayName = await fetchMemberDisplayNameById(assigneeId);
      assigneeName = s(displayName) || assigneeId;
    }

    // ProductIDTag（そのまま表示。空なら "-"）
    const productIdTag = s(pb.productIdTag) || "-";

    // 日時は raw のまま保持（ISO8601/RFC3339 前提）
    const createdAtRaw = s(pb.createdAt);
    const updatedAtRaw = s(pb.updatedAt);

    uiRows.push({
      id: s(pb.id),
      productName: s(pb.productName),
      brandName,
      assigneeName,
      productIdTag,
      createdAt: createdAtRaw,
      // updatedAt が未設定の場合は createdAt に寄せる（欠損対策）
      updatedAt: updatedAtRaw || createdAtRaw,
    });
  }

  return uiRows;
}
