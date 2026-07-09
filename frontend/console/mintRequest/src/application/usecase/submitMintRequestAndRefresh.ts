// frontend/console/mintRequest/src/application/usecase/submitMintRequestAndRefresh.ts

import type { MintDTO } from "../../infrastructure/api/mintRequestApi";
import type { MintQueuedResponse } from "../port/MintRequestRepository";

import {
  postMintRequestHTTP,
  fetchMintByProductionIdHTTP,
} from "../../infrastructure/repository";

/**
 * mint request を送信し、mint を再取得して返す。
 *
 * NOTE:
 * - Backend は同期mint結果ではなく 202 Accepted / QUEUED を返す。
 * - updatedBatch は返らないため、queuedResponse と refreshedMint を返す。
 */
export async function submitMintRequestAndRefresh(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
): Promise<{
  queuedResponse: MintQueuedResponse | null;
  refreshedMint: MintDTO | null;
}> {
  const pid = String(productionId ?? "").trim();
  const tbId = String(tokenBlueprintId ?? "").trim();

  if (!pid || !tbId) {
    return { queuedResponse: null, refreshedMint: null };
  }

  const queuedResponse = await postMintRequestHTTP(
    pid,
    tbId,
    scheduledBurnDate,
  ).catch(() => null);

  const refreshedMint = await fetchMintByProductionIdHTTP(pid).catch(
    () => null,
  );

  return {
    queuedResponse: queuedResponse ?? null,
    refreshedMint: refreshedMint ?? null,
  };
}