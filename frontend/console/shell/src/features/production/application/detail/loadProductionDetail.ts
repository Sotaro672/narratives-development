// frontend/console/shell/src/features/production/application/detail/loadProductionDetail.ts

import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";
import type { ProductBlueprintCategorySnapshot } from "../../../productBlueprint/domain/productBlueprintCategory";
import type { ProductionDetail, ProductionQuantityRow } from "./types";

type ProductionDetailResponse = {
  id: string;
  productBlueprintId: string;
  productName: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;
  brandId: string;
  brandName: string;
  assigneeId: string;
  assigneeName: string;
  models: ProductionQuantityRow[];
  totalQuantity: number;
  printed: boolean;
  printedAt?: string | null;
  createdBy?: string | null;
  createdByName?: string;
  createdAt?: string | null;
  updatedBy?: string | null;
  updatedByName?: string;
  updatedAt?: string | null;
};

function toDate(value?: string | null): Date | null {
  return value ? new Date(value) : null;
}

/**
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

  const repository = new ProductionRepositoryHTTP();
  const response = (await repository.getById(id)) as unknown as ProductionDetailResponse;

  if (!response) {
    return null;
  }

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
    createdById: response.createdBy ?? null,
    createdByName: response.createdByName ?? "",
    createdAt: toDate(response.createdAt),
    updatedById: response.updatedBy ?? null,
    updatedByName: response.updatedByName ?? "",
    updatedAt: toDate(response.updatedAt),
  };
}