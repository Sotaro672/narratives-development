// frontend/console/shell/src/features/mintRequest/application/usecase/submitMintRequestAndRefresh.ts

import type { MintQueuedResponse } from "../port/MintRequestRepository";

import { postMintRequestHTTP } from "../../infrastructure/repository";

/**
 * ミント申請を送信する。
 *
 * Backendは同期的にミントを完了せず、
 * 202 Accepted / QUEUEDを返してミント処理を開始する。
 *
 * ミント情報の再取得はこのUsecaseでは行わず、
 * 呼び出し側のreload処理に任せる。
 */
export async function submitMintRequestAndRefresh(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
): Promise<MintQueuedResponse | null> {
  const normalizedProductionId = productionId.trim();
  const normalizedTokenBlueprintId = tokenBlueprintId.trim();
  const normalizedScheduledBurnDate =
    scheduledBurnDate?.trim() || undefined;

  if (
    !normalizedProductionId ||
    !normalizedTokenBlueprintId
  ) {
    return null;
  }

  return postMintRequestHTTP(
    normalizedProductionId,
    normalizedTokenBlueprintId,
    normalizedScheduledBurnDate,
  );
}