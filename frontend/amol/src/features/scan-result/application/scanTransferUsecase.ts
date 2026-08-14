// frontend/amol/src/features/scan-result/application/scanTransferUsecase.ts

import type { MallScanTransferResponse } from "../../shared/types/scanResult";

export type ScanTransferUsecaseDeps = {
  transferScanPurchased: (input: {
    productId: string;
    operationId: string;
    headers?: HeadersInit;
  }) => Promise<MallScanTransferResponse>;
};

export type RunScanAutoTransferInput = {
  productId: string;
  operationId: string;
  headers?: HeadersInit;
};

export type RunScanAutoTransferResult = {
  transferResult: MallScanTransferResponse;
  transferredAssetId: string;
};

export async function runScanAutoTransfer(
  deps: ScanTransferUsecaseDeps,
  input: RunScanAutoTransferInput,
): Promise<RunScanAutoTransferResult> {
  const productId = input.productId.trim();
  const operationId = input.operationId.trim();

  if (!productId) {
    throw new Error("productId is empty");
  }

  if (!operationId) {
    throw new Error("operationId is empty");
  }

  const transferResult = await deps.transferScanPurchased({
    productId,
    operationId,
    headers: input.headers,
  });

  return {
    transferResult,
    transferredAssetId: transferResult.assetId,
  };
}