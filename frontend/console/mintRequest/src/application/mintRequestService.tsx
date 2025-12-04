// frontend/console/mintRequest/src/application/mintRequestService.tsx

import type { InspectionBatchDTO } from "../infrastructure/api/mintRequestApi";
import {
  fetchInspectionBatchesHTTP,
  fetchProductBlueprintPatchHTTP,
} from "../infrastructure/repository/mintRequestRepositoryHTTP";

/**
 * backend/internal/domain/productBlueprint.Patch に対応する DTO
 */
export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  itemType?: string | null; // Go 側 ItemType（"tops" / "bottoms" など）に対応
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: {
    type?: string | null;
  } | null;
  assigneeId?: string | null;
};

/**
 * MintUsecase 経由の /mint/inspections を叩き、
 * 指定された productionId に対応する InspectionBatchDTO を 1 件返す。
 *
 * ※ /mint/inspections は backend の MintUsecase
 *   （＝ GetModelVariationByID を含む処理）を経由する。
 */
export async function loadInspectionBatchFromMintAPI(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) return null;

  // /mint/inspections を実行（Repository 経由）
  const batches = await fetchInspectionBatchesHTTP();

  // productionId で絞り込み
  const hit =
    batches.find((b) => (b as any).productionId === trimmed) ?? null;

  return hit;
}

/**
 * productBlueprintId から、MintUsecase.GetProductBlueprintPatchByID 経由で
 * ProductBlueprint Patch DTO を取得する。
 */
export async function loadProductBlueprintPatch(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const trimmed = productBlueprintId.trim();
  if (!trimmed) return null;

  return await fetchProductBlueprintPatchHTTP(trimmed);
}

/**
 * ミント申請詳細画面向けの TokenBlueprint を解決する。
 * 現状は API がないため undefined を返す。
 * 必要になれば backend の tokenBlueprint API を呼ぶ方式に置き換える。
 */
export function resolveBlueprintForMintRequest(requestId?: string) {
  return undefined;
}
