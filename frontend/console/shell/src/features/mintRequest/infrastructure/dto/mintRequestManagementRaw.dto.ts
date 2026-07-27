// frontend/console/shell/src/features/mintRequest/infrastructure/dto/mintRequestManagementRaw.dto.ts

import type { InspectionStatus } from "../../domain/inspections";

/**
 * MintRequestQueryService が返す一覧用Raw DTO。
 *
 * 前提:
 * - productionIdとinspectionIdのdocIdは同一
 * - フロントではproductionIdを正とする
 * - id / inspectionId / casing fallback / old DTO名は扱わない
 * - createdByとrequestedByは相互補完しない
 */
export type MintRequestManagementRawDTO = {
  productionId: string;

  tokenBlueprintId?: string | null;
  tokenName?: string | null;
  productName?: string | null;

  mintQuantity?: number | null;
  productionQuantity?: number | null;

  inspectionStatus?: InspectionStatus | string | null;

  createdBy?: string | null;
  createdByName?: string | null;

  requestedBy?: string | null;
  requestedByName?: string | null;

  mintedAt?: string | null;

  mint?: {
    id?: string | null;

    brandId?: string | null;
    tokenBlueprintId?: string | null;

    products?: string[] | null;
    status?: string | null;

    createdAt?: string | null;
    createdBy?: string | null;
    requestedBy?: string | null;

    mintedAt?: string | null;
    scheduledBurnDate?: string | null;

    onChainTxSignature?: string | null;
  } | null;
};