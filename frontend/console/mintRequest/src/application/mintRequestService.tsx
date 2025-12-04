// frontend/console/mintRequest/src/application/mintRequestService.tsx

import type { InspectionBatchDTO } from "../infrastructure/api/mintRequestApi";
import { fetchInspectionBatches } from "../infrastructure/api/mintRequestApi";

/**
 * MintUsecase 経由の /mint/inspections を叩き、
 * 指定された productionId に対応する InspectionBatchDTO を 1 件返す。
 *
 * ※ /mint/inspections は backend の MintUsecase（＝ GetModelVariationByID を含む処理）を経由する。
 */
export async function loadInspectionBatchFromMintAPI(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) {
    return null;
  }

  // /mint/inspections を実行
  const batches = await fetchInspectionBatches();

  // productionId で絞り込み
  const hit =
    batches.find((b) => (b as any).productionId === trimmed) ?? null;

  return hit;
}

/**
 * ミント申請詳細画面向けの TokenBlueprint を解決する。
 * 現在は API がないため undefined を返す。
 * 必要になれば backend の tokenBlueprint API を呼ぶ方式に置き換える。
 */
export function resolveBlueprintForMintRequest(
  requestId?: string,
) {
  return undefined;
}
