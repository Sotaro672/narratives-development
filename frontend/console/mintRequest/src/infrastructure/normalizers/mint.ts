// frontend/console/mintRequest/src/infrastructure/normalizers/mint.ts

import type { MintDTO } from "../api/mintRequestApi";

/**
 * MintDTO normalize
 * - tokenBlueprintId / brandId は lowerCamel を正として扱う
 * - inspectionId は productionId と同一視される実装が残り得るため最小限の互換を維持
 */
export function normalizeMintDTO(v: any): MintDTO {
  const obj: any = { ...(v ?? {}) };

  // id
  obj.id = obj.id ?? "";

  // ✅ tokenBlueprintId / brandId は lowerCamel を正として扱う
  obj.brandId = obj.brandId ?? "";
  obj.tokenBlueprintId = obj.tokenBlueprintId ?? "";

  // inspectionId（productionId と同一視される可能性の互換）
  obj.inspectionId = obj.inspectionId ?? obj.productionId ?? obj.ProductionID ?? "";

  obj.createdAt = obj.createdAt ?? null;
  obj.createdBy = obj.createdBy ?? "";
  obj.createdByName = obj.createdByName ?? null;

  // tokenName（あれば）
  obj.tokenName = obj.tokenName ?? null;

  obj.minted =
    typeof obj.minted === "boolean" ? obj.minted : Boolean(obj.mintedAt ?? null);
  obj.mintedAt = obj.mintedAt ?? null;

  obj.scheduledBurnDate = obj.scheduledBurnDate ?? null;
  obj.onChainTxSignature = obj.onChainTxSignature ?? null;

  return obj as MintDTO;
}
