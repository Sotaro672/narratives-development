// frontend/console/mintRequest/src/infrastructure/dto/mintRequestManagementRaw.dto.ts

import type { InspectionStatus } from "../../domain/entity/inspections";

/**
 * MintRequestQueryService が返す “一覧用 Raw DTO” をそのまま表現。
 *
 * 前提:
 * - productionId と inspectionId の docId は同一
 * - フロントでは productionId を正とする
 * - id / inspectionId / casing fallback / old DTO 名は扱わない
 */
export type MintRequestManagementRawDTO = {
  productionId: string;

  // list fields
  tokenName?: string | null;
  productName?: string | null;

  mintQuantity?: number | null;
  productionQuantity?: number | null;

  inspectionStatus?: InspectionStatus | string | null;

  requestedBy?: string | null;
  requestedByName?: string | null;
  createdByName?: string | null;

  mintedAt?: string | null;

  tokenBlueprintId?: string | null;

  productBlueprintId?: string | null;
  scheduledBurnDate?: string | null;
  minted?: boolean | null;

  // raw sub docs
  mint?: {
    tokenBlueprintId?: string | null;
    tokenName?: string | null;
    mintedAt?: string | null;
    createdBy?: string | null;
  } | null;

  inspection?: {
    status?: InspectionStatus | string | null;
    productName?: string | null;
  } | null;
};

/**
 * レスポンスが `{ items: [...] }` の場合の wrapper
 */
export type MintRequestManagementRawResponseDTO = {
  items?: MintRequestManagementRawDTO[] | null;
};