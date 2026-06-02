// frontend/console/production/src/application/create/ProductionCreateService.ts

import type { Production } from "./ProductionCreateTypes";
import type { ProductionRepository } from "./ProductionCreateRepository";

// ======================================================================
// Application Service for Production Create
// ======================================================================

export type ProductionQuantityInput = {
  modelId: string;
  quantity: number;
};

/**
 * Build Production request payload based on the Firestore absolute schema
 * (models: [{ ModelID, Quantity }], printed boolean, printedAt/printedBy optional).
 */
export function buildProductionRequest(params: {
  productBlueprintId: string;
  assigneeId: string;
  creatorUid: string;
  quantities: ProductionQuantityInput[];
  nowIso?: () => string;
}): Production {
  const {
    productBlueprintId,
    assigneeId,
    creatorUid,
    quantities,
    nowIso = () => new Date().toISOString(),
  } = params;

  const createdAt = nowIso();

  return {
    id: "",
    productBlueprintId,
    assigneeId,

    // absolute schema: Firestore doc stores keys as ModelID / Quantity
    models: quantities.map((q) => ({
      ModelID: q.modelId,
      Quantity: q.quantity,
    })),

    printed: false,
    printedAt: null,
    printedBy: null,

    // createdBy must store Firebase Auth UID, not members docId
    createdBy: creatorUid,
    createdAt,

    // optional (table shows updatedAt exists after print/update)
    updatedAt: null,
  } as unknown as Production;
}

export function buildProductionPayload(params: {
  productBlueprintId: string;
  assigneeId: string;
  rows: ProductionQuantityInput[];

  // Firebase Auth UID
  currentMemberUid: string | null;

  nowIso?: () => string;
}): Production {
  const { productBlueprintId, assigneeId, rows, currentMemberUid, nowIso } =
    params;

  return buildProductionRequest({
    productBlueprintId,
    assigneeId,
    creatorUid: currentMemberUid ?? "",
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