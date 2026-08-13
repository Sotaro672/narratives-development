// frontend/console/shell/src/features/production/application/productionDetailService.ts

import type {
  ProductionDetail,
  UpdateProductionRequest,
} from "../../../shared/types/production";
import { fetchProductionDetail } from "../infrastructure/api/productionDetailApi";
import { ProductionRepositoryHTTP } from "../infrastructure/http/productionRepositoryHTTP";

// ======================================================================
// Production Detail Load
// ======================================================================

export async function loadProductionDetail(productionId: string): Promise<ProductionDetail | null> {
  const id = productionId.trim();
  if (!id) {
    return null;
  }

  return fetchProductionDetail(id);
}

// ======================================================================
// Production Detail Update
// ======================================================================

export async function updateProductionDetail(
  productionId: string,
  request: UpdateProductionRequest,
): Promise<ProductionDetail | null> {
  const id = productionId.trim();
  if (!id) {
    throw new Error("productionId is required");
  }

  const assigneeId = request.assigneeId.trim();
  if (!assigneeId) {
    throw new Error("assigneeId is required");
  }

  const models = request.models
    .map((row) => {
      const modelId = row.modelId.trim();
      const quantityNumber = Number(row.quantity);
      const quantity = Number.isFinite(quantityNumber) ? Math.max(0, Math.floor(quantityNumber)) : 0;

      return {
        modelId,
        quantity,
      };
    })
    .filter((model) => model.modelId !== "");

  const repository = new ProductionRepositoryHTTP();
  await repository.update(id, {
    assigneeId,
    models,
  });

  return loadProductionDetail(id);
}