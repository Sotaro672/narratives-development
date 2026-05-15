// frontend/console/mintRequest/src/application/usecase/submitMintRequestAndRefresh.ts

import type { InspectionBatchDTO } from "../../domain/entity/inspections";
import type { MintDTO } from "../../infrastructure/api/mintRequestApi";

import {
  postMintRequestHTTP,
  fetchMintByProductionIdHTTP,
} from "../../infrastructure/repository";

/**
 * mint request を送信し、inspection と mint を再取得して返す。
 */
export async function submitMintRequestAndRefresh(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
): Promise<{
  updatedBatch: InspectionBatchDTO | null;
  refreshedMint: MintDTO | null;
}> {
  const pid = String(productionId ?? "").trim();
  const tbId = String(tokenBlueprintId ?? "").trim();

  if (!pid || !tbId) {
    return { updatedBatch: null, refreshedMint: null };
  }

  const updatedBatch = await postMintRequestHTTP(
    pid,
    tbId,
    scheduledBurnDate,
  ).catch(() => null);

  const refreshedMint = await fetchMintByProductionIdHTTP(pid).catch(
    () => null,
  );

  return {
    updatedBatch: updatedBatch ?? null,
    refreshedMint: refreshedMint ?? null,
  };
}