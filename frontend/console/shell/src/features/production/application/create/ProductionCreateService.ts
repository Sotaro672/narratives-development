// frontend/console/shell/src/features/production/application/create/ProductionCreateService.ts

import type { Production } from "../../../../shared/types/production";

// ======================================================================
// Port: ProductionRepository
// ======================================================================

export interface ProductionRepository {
  create(payload: Production): Promise<Production>;
}

// ======================================================================
// Production Create
// ======================================================================

export type ProductionQuantityInput = {
  modelId: string;
  quantity: number;
};

/**
 * POST /productions 用payloadを構築する。
 *
 * createdAt / printedAt / printedBy / updatedBy / updatedAt は
 * frontendでは生成せず、backend側の責務とする。
 */
export function buildProductionPayload(params: {
  productBlueprintId: string;
  assigneeId: string;
  rows: ProductionQuantityInput[];
  currentMemberUid: string | null;
}): Production {
  const {
    productBlueprintId,
    assigneeId,
    rows,
    currentMemberUid,
  } = params;

  return {
    id: "",
    productBlueprintId,
    assigneeId,
    models: rows.map((row) => ({
      modelId: row.modelId,
      quantity: row.quantity,
    })),
    printed: false,
    createdBy: currentMemberUid,
  };
}