// frontend/console/shell/src/features/production/application/productionDetailService.ts

import type { ProductBlueprintCategorySnapshot } from "../../productBlueprint/domain/productBlueprintCategory";
import { fetchProductionDetail } from "../infrastructure/api/productionDetailApi";
import { ProductionRepositoryHTTP } from "../infrastructure/http/productionRepositoryHTTP";
import type {
  ProductionQuantityInput,
  ProductionQuantityRow,
} from "./productionQuantityRow";

// ======================================================================
// Production Detail
// ======================================================================

export type ProductionDetail = {
  id: string;
  productBlueprintId: string;
  productName: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;
  brandId: string;
  brandName: string;
  assigneeId: string;
  assigneeName: string;
  printed: boolean;
  models: ProductionQuantityRow[];
  totalQuantity: number;
  printedAt: Date | null;
  printedBy?: string | null;
  printedByName: string;
  createdBy?: string | null;
  createdByName: string;
  createdAt: Date | null;
  updatedBy?: string | null;
  updatedByName: string;
  updatedAt: Date | null;
};

function toDate(value?: string | null): Date | null {
  return value ? new Date(value) : null;
}

// ======================================================================
// Production Detail Load
// ======================================================================

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
    printedBy: response.printedBy ?? null,
    printedByName: response.printedByName ?? "",
    createdBy: response.createdBy ?? null,
    createdByName: response.createdByName ?? "",
    createdAt: toDate(response.createdAt),
    updatedBy: response.updatedBy ?? null,
    updatedByName: response.updatedByName ?? "",
    updatedAt: toDate(response.updatedAt),
  };
}

// ======================================================================
// Production Detail Update
// ======================================================================

export async function updateProductionDetail(params: {
  productionId: string;
  rows: ProductionQuantityInput[];
  assigneeId?: string | null;
}): Promise<ProductionDetail | null> {
  const { productionId, rows, assigneeId } = params;
  const id = productionId.trim();

  if (!id) {
    throw new Error("productionId is required");
  }

  const normalizedAssigneeId = String(assigneeId ?? "").trim();

  if (!normalizedAssigneeId) {
    throw new Error("assigneeId is required");
  }

  const models = rows
    .map((row) => {
      const modelId = String(row.modelId ?? "").trim();
      const quantityNumber = Number(row.quantity);
      const quantity = Number.isFinite(quantityNumber)
        ? Math.max(0, Math.floor(quantityNumber))
        : 0;

      return {
        modelId,
        quantity,
      };
    })
    .filter((model) => model.modelId !== "");

  const repository = new ProductionRepositoryHTTP();

  await repository.update(id, {
    assigneeId: normalizedAssigneeId,
    models,
  });

  return loadProductionDetail(id);
}