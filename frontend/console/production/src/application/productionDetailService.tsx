// frontend/console/production/src/application/productionDetailService.tsx

import type {
  Production,
  ProductionStatus,
} from "../../../shell/src/shared/types/production";
import { ProductionRepositoryHTTP } from "../infrastructure/http/productionRepositoryHTTP";
import { listProductionsHTTP } from "../infrastructure/query/productionQuery";

/**
 * 詳細表示用型
 */
export type ProductionDetail = Production & {
  totalQuantity: number;
  assigneeName?: string;
  productBlueprintName?: string;
  brandName?: string;
};

export async function loadProductionDetail(
  productionId: string,
): Promise<ProductionDetail | null> {
  if (!productionId) return null;

  const repo = new ProductionRepositoryHTTP();
  const raw = (await repo.getById(productionId)) as any;
  if (!raw) return null;

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

  let detail: ProductionDetail = {
    ...(raw as Production),
    id: raw.id ?? raw.ID ?? "",
    productBlueprintId: blueprintId,
    productBlueprintName:
      raw.productBlueprintName ??
      raw.ProductBlueprintName ??
      blueprintId,
    brandName:
      raw.brandName ??
      raw.BrandName ??
      raw.brand ??
      raw.Brand ??
      "",
    assigneeId: raw.assigneeId ?? raw.AssigneeID ?? "",
    assigneeName: raw.assigneeName ?? raw.AssigneeName ?? "",
    status: (raw.status ?? raw.Status ?? "") as ProductionStatus,
    printedAt: raw.printedAt ?? raw.PrintedAt ?? null,
    createdAt: raw.createdAt ?? raw.CreatedAt ?? null,
    updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? null,
    models: rawModels,
    totalQuantity,
  };

  // =====================================================
  // ★ 一覧データからの名前解決（productBlueprintName / brandName / assigneeName）
  // =====================================================
  try {
    if (
      !detail.productBlueprintName ||
      detail.productBlueprintName === detail.productBlueprintId ||
      !detail.brandName ||
      !detail.assigneeName
    ) {
      const listItems = await listProductionsHTTP();

      const match = (listItems as any[]).find((item) => {
        const itemId = item.id ?? item.ID ?? "";
        const itemBlueprintId =
          item.productBlueprintId ?? item.ProductBlueprintID ?? "";
        return (
          itemId === detail.id ||
          (itemBlueprintId &&
            itemBlueprintId === detail.productBlueprintId)
        );
      });

      if (match) {
        const resolvedBlueprintName =
          match.productBlueprintName ??
          match.ProductBlueprintName ??
          detail.productBlueprintId;

        const resolvedBrandName =
          match.brandName ?? match.BrandName ?? "";

        const resolvedAssigneeName =
          match.assigneeName ?? match.AssigneeName ?? "";

        detail = {
          ...detail,
          productBlueprintName:
            detail.productBlueprintName &&
            detail.productBlueprintName !== detail.productBlueprintId
              ? detail.productBlueprintName
              : resolvedBlueprintName,
          brandName: detail.brandName || resolvedBrandName,
          assigneeName: detail.assigneeName || resolvedAssigneeName,
        };
      }
    }
  } catch (e) {
    console.warn(
      "[productionDetailService] failed to resolve names from list:",
      e,
    );
  }

  console.log("[productionDetailService] loaded detail:", detail);
  return detail;
}
