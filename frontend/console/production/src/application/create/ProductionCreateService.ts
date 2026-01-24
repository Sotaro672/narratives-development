import type { Production } from "./ProductionCreateTypes";
import type { ProductionRepository } from "./ProductionCreateRepository";

// ======================================================================
// Application Service for Production Create
// ======================================================================

export type ProductionQuantityInput = {
  modelVariationId: string;
  quantity: number;
};

export function buildProductionRequest(params: {
  productBlueprintId: string;
  assigneeId: string;
  creatorId: string;
  quantities: ProductionQuantityInput[];
  nowIso?: () => string;
}): Production {
  const {
    productBlueprintId,
    assigneeId,
    creatorId,
    quantities,
    nowIso = () => new Date().toISOString(),
  } = params;

  return {
    id: "",
    productBlueprintId,
    assigneeId,
    models: quantities.map((q) => ({
      modelId: q.modelVariationId,
      quantity: q.quantity,
    })),
    status: "planned",
    createdBy: creatorId,
    createdAt: nowIso(),
  };
}

export function buildProductionPayload(params: {
  productBlueprintId: string;
  assigneeId: string;
  rows: ProductionQuantityInput[];
  currentMemberId: string | null;
  nowIso?: () => string;
}): Production {
  const { productBlueprintId, assigneeId, rows, currentMemberId, nowIso } =
    params;

  return buildProductionRequest({
    productBlueprintId,
    assigneeId,
    creatorId: currentMemberId ?? "",
    quantities: rows,
    nowIso,
  });
}

// ======================================================================
// Usecase execution
// ======================================================================
// 注意: repo は application 外（composition root / DI）で注入する
export async function createProduction(
  repo: ProductionRepository,
  payload: Production,
): Promise<Production> {
  return await repo.create(payload);
}
