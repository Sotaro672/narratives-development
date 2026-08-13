// frontend/console/shell/src/features/production/application/detail/loadProductionDetail.ts

import { fetchProductionDetail } from "../../infrastructure/api/productionDetailApi";
import type { ProductionDetail } from "./types";

function toDate(value?: string | null): Date | null {
  return value ? new Date(value) : null;
}

/**
 * Production詳細取得。
 *
 * GET /productions/{id} の Production Detail BFF response を正とする。
 * ProductBlueprint / ModelVariation / Production一覧からの追加補完は行わない。
 */
export async function loadProductionDetail(
  productionId: string,
): Promise<ProductionDetail | null> {
  const id = productionId.trim();

  if (!id) {
    return null;
  }

  const response = await fetchProductionDetail(id);

  return {
    id: response.id,
    productBlueprintId: response.productBlueprintId,
    productName: response.productName,
    productBlueprintCategory: response.productBlueprintCategory,
    brandId: response.brandId,
    brandName: response.brandName,
    assigneeId: response.assigneeId,
    assigneeName: response.assigneeName,
    printed: response.printed,
    models: response.models,
    totalQuantity: response.totalQuantity,
    printedAt: toDate(response.printedAt),
    createdBy: response.createdBy ?? null,
    createdByName: response.createdByName ?? "",
    createdAt: toDate(response.createdAt),
    updatedBy: response.updatedBy ?? null,
    updatedByName: response.updatedByName ?? "",
    updatedAt: toDate(response.updatedAt),
  };
}