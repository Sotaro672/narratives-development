// frontend/console/shell/src/features/production/application/create/ProductionCreateService.ts

import type {
  Production,
} from "../../../../shared/types/production";

// ======================================================================
// Port: ProductionRepository
// ======================================================================
// Application層はI/Oの詳細を知らない。
// Infrastructure層がこのPortを実装する。
export interface ProductionRepository {
  create(
    payload: Production,
  ): Promise<Production>;
}

// ======================================================================
// Application Service for Production Create
// ======================================================================

export type ProductionQuantityInput = {
  modelId: string;
  quantity: number;
};

/**
 * BackendのProduction作成APIへ送信するpayloadを構築する。
 *
 * models:
 * [
 *   {
 *     modelId: string;
 *     quantity: number;
 *   }
 * ]
 */
export function buildProductionRequest(
  params: {
    productBlueprintId: string;
    assigneeId: string;
    creatorUid: string;
    quantities: ProductionQuantityInput[];
    nowIso?: () => string;
  },
): Production {
  const {
    productBlueprintId,
    assigneeId,
    creatorUid,
    quantities,
    nowIso = () =>
      new Date().toISOString(),
  } = params;

  const createdAt =
    nowIso();

  return {
    id: "",

    productBlueprintId,

    assigneeId,

    models:
      quantities.map(
        (quantity) => ({
          modelId:
            quantity.modelId,

          quantity:
            quantity.quantity,
        }),
      ),

    printed: false,

    printedAt: null,

    printedBy: null,

    // createdByにはmembersのdocument IDではなく
    // Firebase Auth UIDを保存する。
    createdBy:
      creatorUid,

    createdAt,

    updatedBy: null,

    updatedAt: null,
  };
}

export function buildProductionPayload(
  params: {
    productBlueprintId: string;
    assigneeId: string;
    rows: ProductionQuantityInput[];

    // Firebase Auth UID
    currentMemberUid:
      string | null;

    nowIso?: () => string;
  },
): Production {
  const {
    productBlueprintId,
    assigneeId,
    rows,
    currentMemberUid,
    nowIso,
  } = params;

  return buildProductionRequest({
    productBlueprintId,

    assigneeId,

    creatorUid:
      currentMemberUid ?? "",

    quantities:
      rows,

    nowIso,
  });
}