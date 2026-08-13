// frontend/console/shell/src/features/production/application/create/ProductionCreateService.ts

// ======================================================================
// Production Create DTO
// ======================================================================

export type ProductionQuantityInput = {
  modelId: string;
  quantity: number;
};

export type CreateProductionRequest = {
  productBlueprintId: string;
  assigneeId: string;
  models: ProductionQuantityInput[];
};

// ======================================================================
// Port: ProductionRepository
// ======================================================================

export interface ProductionRepository {
  create(payload: CreateProductionRequest): Promise<unknown>;
}

// ======================================================================
// Production Create
// ======================================================================

export function buildProductionPayload(params: {
  productBlueprintId: string;
  assigneeId: string;
  rows: ProductionQuantityInput[];
}): CreateProductionRequest {
  const { productBlueprintId, assigneeId, rows } = params;

  return {
    productBlueprintId,
    assigneeId,
    models: rows.map((row) => ({
      modelId: row.modelId,
      quantity: row.quantity,
    })),
  };
}