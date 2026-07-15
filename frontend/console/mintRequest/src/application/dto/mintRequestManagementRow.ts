// frontend/console/mintRequest/src/application/dto/mintRequestManagementRow.ts

import type { InspectionStatus } from "../../domain/inspections";

/**
 * MintRequestQueryService が返す “一覧用 DTO”。
 *
 * 前提:
 * - productions / inspections / mints の docId はすべて同一
 * - productionId を正とする
 * - id / inspectionId / mintId は主キーとして扱わない
 */
export type MintRequestManagementRowDTO = {
  productionId: string;

  tokenName?: string | null;
  productName?: string | null;

  mintQuantity?: number | null;
  productionQuantity?: number | null;

  inspectionStatus?: InspectionStatus | string | null;

  requestedBy?: string | null;
  requestedByName?: string | null;
  createdByName?: string | null;

  mintedAt?: string | null;
  minted?: boolean | null;

  tokenBlueprintId?: string | null;
  productBlueprintId?: string | null;
  scheduledBurnDate?: string | null;
};