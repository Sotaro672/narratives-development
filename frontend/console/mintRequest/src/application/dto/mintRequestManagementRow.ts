// frontend/console/mintRequest/src/application/dto/mintRequestManagementRow.ts

import type { InspectionStatus } from "../../domain/entity/inspections";

/**
 * MintRequestQueryService が返す “一覧用 DTO” を想定（フィールド名の揺れを吸収するための型）
 *
 * - 現行 backend (ProductionInspectionMintDTO) 想定:
 *   - mintQuantity / productionQuantity を優先
 *   - mintedAt / tokenBlueprintId は top-level or mint 配下の両方を許容
 *   - inspectionStatus / productName も top-level or inspection 配下を許容
 */
export type MintRequestManagementRowDTO = {
  id?: string;
  productionId?: string;
  inspectionId?: string;

  // list fields (preferred)
  tokenName?: string | null;
  productName?: string | null;
  mintQuantity?: number | null;
  productionQuantity?: number | null;
  inspectionStatus?: InspectionStatus | string | null;

  // legacy/alt fields (fallback)
  totalPassed?: number | null;
  quantity?: number | null;

  // requestedBy = mint.createdBy
  requestedBy?: string | null;
  requestedByName?: string | null;
  createdByName?: string | null;

  mintedAt?: string | null;

  // token blueprint (optional)
  tokenBlueprintId?: string | null;

  // raw sub docs (optional)
  mint?: {
    tokenBlueprintId?: string | null;
    tokenBlueprintID?: string | null; // casing fallback
    tokenBlueprint?: string | null; // older list DTO name
    tokenName?: string | null;
    mintedAt?: string | null;
    createdBy?: string | null;
  } | null;

  inspection?: {
    status?: InspectionStatus | string | null;
    productName?: string | null;
    totalPassed?: number | null;
    quantity?: number | null;
  } | null;
};
