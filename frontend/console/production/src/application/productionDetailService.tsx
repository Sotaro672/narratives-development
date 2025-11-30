// frontend/console/production/src/application/productionDetailService.tsx

import type {
  Production,
  ProductionStatus,
} from "../../../shell/src/shared/types/production";
import { ProductionRepositoryHTTP } from "../infrastructure/http/productionRepositoryHTTP";

/**
 * 詳細表示用型
 * - totalQuantity 付与
 * - assigneeName / productBlueprintName / brandName を含む
 */
export type ProductionDetail = Production & {
  totalQuantity: number;
  assigneeName?: string;
  productBlueprintName?: string;
  brandName?: string; // ★ 新規追加
};

/**
 * Production 1件取得
 * - backend フィールド名揺れに対応
 * - models から totalQuantity を算出
 */
export async function loadProductionDetail(
  productionId: string,
): Promise<ProductionDetail | null> {
  if (!productionId) return null;

  const repo = new ProductionRepositoryHTTP();

  // 取得
  const raw = (await repo.getById(productionId)) as any;
  if (!raw) return null;

  // --- models 正規化 ---
  const rawModels = Array.isArray(raw.models)
    ? raw.models
    : Array.isArray(raw.Models)
    ? raw.Models
    : [];

  const totalQuantity = rawModels.reduce(
    (sum: number, m: any) => sum + (m?.quantity ?? m?.Quantity ?? 0),
    0,
  );

  const blueprintId =
    raw.productBlueprintId ?? raw.ProductBlueprintID ?? "";

  // --- brandName の吸収（一覧と同様） ---
  const brandName =
    raw.brandName ??
    raw.BrandName ??
    raw.brand ??
    raw.Brand ??
    "";

  const detail: ProductionDetail = {
    ...(raw as Production),

    id: raw.id ?? raw.ID ?? "",
    productBlueprintId: blueprintId,

    // productBlueprintName（なければ ID）
    productBlueprintName:
      raw.productBlueprintName ??
      raw.ProductBlueprintName ??
      blueprintId,

    // brandName（一覧と揃える）
    brandName,

    assigneeId: raw.assigneeId ?? raw.AssigneeID ?? "",
    assigneeName: raw.assigneeName ?? raw.AssigneeName ?? "",

    status: (raw.status ?? raw.Status ?? "") as ProductionStatus,

    printedAt: raw.printedAt ?? raw.PrintedAt ?? null,
    createdAt: raw.createdAt ?? raw.CreatedAt ?? null,
    updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? null,

    models: rawModels,
    totalQuantity,
  };

  console.log("[productionDetailService] loaded detail:", detail);

  return detail;
}
