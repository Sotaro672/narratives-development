// frontend/console/productBlueprint/src/infrastructure/query/productBlueprintQuery.ts

import { listProductBlueprintsHTTP } from "../repository/productBlueprintRepositoryHTTP";

export type ProductBlueprintManagementRow = {
  id: string;
  productName: string;

  /**
   * backend 側で name 解決済みを返す前提
   */
  brandName: string;
  assigneeName: string;

  productIdTag: string;

  /**
   * backend は ISO8601/RFC3339 の日時文字列を返す。
   * この層では表示整形を行わず、raw の日時文字列を保持して presentation に渡す。
   */
  createdAt: string; // ISO8601/RFC3339 datetime string
  updatedAt: string; // ISO8601/RFC3339 datetime string
};

// backend /product-blueprints のレスポンス想定（backend 側で name 解決済み）
type RawProductBlueprintListRow = {
  id?: string;
  productName?: string;

  /**
   * backend 側で解決済みを返す（フロントでの brandId/assigneeId 解決は不要）
   */
  brandName?: string | null;
  assigneeName?: string | null;

  // backend の JSON は "productIdTag": "QRコード" などの文字列を直接返す想定
  productIdTag?: string | null;

  // backend は createdAt/updatedAt を ISO8601/RFC3339 の文字列で返す前提
  createdAt?: string | null;
  updatedAt?: string | null;
};

function s(v: unknown): string {
  return v == null ? "" : String(v).trim();
}

/**
 * backend から商品設計一覧を取得し、
 * backend 側で解決済みの
 * - brandName
 * - assigneeName
 * をそのまま UI Row にマッピングする。
 *
 * 修正案B:
 * - createdAt/updatedAt の「表示整形」はこの層で行わない（時刻情報を保持）
 */
export async function fetchProductBlueprintManagementRows(): Promise<ProductBlueprintManagementRow[]> {
  const list = (await listProductBlueprintsHTTP()) as RawProductBlueprintListRow[];

  const uiRows: ProductBlueprintManagementRow[] = [];

  for (const pb of list) {
    const brandName = s(pb.brandName);
    const assigneeName = s(pb.assigneeName) || "-";

    const productIdTag = s(pb.productIdTag) || "-";

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
