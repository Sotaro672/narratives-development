// frontend/console/shell/src/features/production/application/detail/updateProductionDetail.ts

import type { ProductionDetail } from "./types";
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";
import { loadProductionDetail } from "./loadProductionDetail";

export type ProductionQuantityInput = {
  modelId: string;
  quantity: number;
};

/* ---------------------------------------------------------
 * Production 更新リクエスト（usecase）
 * --------------------------------------------------------- */
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

  const modelsPayload = (Array.isArray(rows) ? rows : [])
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
    models: modelsPayload,
  });

  return loadProductionDetail(id);
}