// frontend/mall/src/features/scan-result/application/scanTransferUsecase.ts

import type {
  MallScanTransferResponse,
  PreviewState,
} from "../../shared/types/scanResult";
import type { ScanOwnershipResult } from "./scanOwnershipUsecase";

export const TRANSFER_PREVIEW_RECOVERY_ATTEMPTS = 6;
const TRANSFER_PREVIEW_RECOVERY_BASE_DELAY_MS = 700;

export type ReturnInProgressOpenedErrorLike = Error & {
  avatarId: string;
  productId: string;
  matchedOrderId: string;
  matchedItemIndex: number;
};

export type ScanTransferUsecaseDeps = {
  transferScanPurchased: (input: {
    productId: string;
    operationId: string;
    headers?: HeadersInit;
  }) => Promise<MallScanTransferResponse>;
  loadPreviewState: (productId: string) => Promise<PreviewState>;
  checkOwnershipByAssetId: (input: {
    assetId: string;
    headers: HeadersInit;
    retryAfterTransfer?: boolean;
  }) => Promise<ScanOwnershipResult>;
  isReturnInProgressOpenedError: (
    error: unknown,
  ) => error is ReturnInProgressOpenedErrorLike;
  getOrCreateTransferOperationId: (productId: string) => string;
  clearStoredTransferOperationId: (
    productId: string,
    operationId: string,
  ) => void;
  wait?: (ms: number) => Promise<void>;
};

export type RecoverScanTransferInput = {
  productId: string;
  assetId: string;
  operationId: string;
};

export type RecoverScanTransferResult = {
  recovered: boolean;
  previewState: PreviewState | null;
  transferResult: MallScanTransferResponse | null;
  operationId: string;
};

export type ExecuteScanTransferInput = {
  productId: string;
  assetId: string;
  headers: HeadersInit;
  operationId?: string;
};

export type ExecuteScanTransferResult = {
  transferResult: MallScanTransferResponse | null;
  previewState: PreviewState | null;
  ownedByWallet: boolean | null;
  ownedByWalletError: string | null;
  transferError: string | null;
  transferModalError: string | null;
  shouldOpenTransferModal: boolean;
  operationId: string;
  recovered: boolean;
};

function defaultWait(ms: number): Promise<void> {
  return new Promise((resolve) => {
    globalThis.setTimeout(resolve, ms);
  });
}

function toErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function readErrorStatus(error: unknown): number | null {
  if (
    typeof error !== "object" ||
    error === null ||
    !("status" in error)
  ) {
    return null;
  }

  const status = (error as { status?: unknown }).status;
  return typeof status === "number" && Number.isFinite(status)
    ? status
    : null;
}

export function isRetryableScanTransferError(
  error: unknown,
  isReturnInProgressOpenedError: (
    error: unknown,
  ) => error is ReturnInProgressOpenedErrorLike,
): boolean {
  if (isReturnInProgressOpenedError(error)) {
    return false;
  }

  const status = readErrorStatus(error);

  if (status !== null) {
    return status === 408 || status >= 500;
  }

  return error instanceof TypeError;
}

// Recovery path does not reconstruct order/item identity.
// Therefore this result must never be used to enable Avatar Review.
export function createRecoveredScanTransferResult(
  previewState: PreviewState,
  assetId: string,
): MallScanTransferResponse | null {
  const normalizedAssetId = assetId.trim();

  if (!normalizedAssetId) {
    return null;
  }

  return {
    avatarId: previewState.raw.owner?.avatarId ?? "",
    productId: previewState.raw.productId,
    matched: true,
    txSignature: "",
    updatedToAddress: true,
    assetId: normalizedAssetId,
  };
}

export async function recoverScanTransferAfterOwnershipConfirmed(
  deps: ScanTransferUsecaseDeps,
  input: RecoverScanTransferInput,
): Promise<RecoverScanTransferResult> {
  const normalizedProductId = input.productId.trim();
  const normalizedAssetId = input.assetId.trim();
  const operationId = input.operationId.trim();

  if (!normalizedProductId || !normalizedAssetId) {
    return {
      recovered: false,
      previewState: null,
      transferResult: null,
      operationId,
    };
  }

  const wait = deps.wait ?? defaultWait;

  for (
    let attempt = 1;
    attempt <= TRANSFER_PREVIEW_RECOVERY_ATTEMPTS;
    attempt += 1
  ) {
    try {
      const previewState =
        await deps.loadPreviewState(normalizedProductId);

      const transferResult = createRecoveredScanTransferResult(
        previewState,
        normalizedAssetId,
      );

      if (transferResult) {
        if (operationId) {
          deps.clearStoredTransferOperationId(
            normalizedProductId,
            operationId,
          );
        }

        return {
          recovered: true,
          previewState,
          transferResult,
          operationId: "",
        };
      }
    } catch {
      // Transfer 自体は完了している可能性があるため、
      // Preview BFF の反映を待って再試行する。
    }

    if (attempt < TRANSFER_PREVIEW_RECOVERY_ATTEMPTS) {
      await wait(
        TRANSFER_PREVIEW_RECOVERY_BASE_DELAY_MS * attempt,
      );
    }
  }

  return {
    recovered: false,
    previewState: null,
    transferResult: null,
    operationId,
  };
}

export async function executeScanTransfer(
  deps: ScanTransferUsecaseDeps,
  input: ExecuteScanTransferInput,
): Promise<ExecuteScanTransferResult> {
  const normalizedProductId = input.productId.trim();
  const normalizedAssetId = input.assetId.trim();

  if (!normalizedProductId) {
    return {
      transferResult: null,
      previewState: null,
      ownedByWallet: null,
      ownedByWalletError: null,
      transferError: null,
      transferModalError: null,
      shouldOpenTransferModal: false,
      operationId: "",
      recovered: false,
    };
  }

  let operationId =
    input.operationId?.trim() ||
    deps.getOrCreateTransferOperationId(normalizedProductId);

  try {
    const transferResult = await deps.transferScanPurchased({
      productId: normalizedProductId,
      operationId,
      headers: input.headers,
    });

    if (operationId) {
      deps.clearStoredTransferOperationId(
        normalizedProductId,
        operationId,
      );
      operationId = "";
    }

    return {
      transferResult,
      previewState: null,
      ownedByWallet: null,
      ownedByWalletError: null,
      transferError: null,
      transferModalError: null,
      shouldOpenTransferModal: transferResult.matched,
      operationId,
      recovered: false,
    };
  } catch (caughtError) {
    if (deps.isReturnInProgressOpenedError(caughtError)) {
      const blockedResult: MallScanTransferResponse = {
        avatarId: caughtError.avatarId,
        productId:
          caughtError.productId || normalizedProductId,
        matched: false,
        matchedOrderId: caughtError.matchedOrderId,
        matchedItemIndex: caughtError.matchedItemIndex,
        txSignature: "",
        updatedToAddress: false,
        assetId: normalizedAssetId,
      };

      if (operationId) {
        deps.clearStoredTransferOperationId(
          normalizedProductId,
          operationId,
        );
        operationId = "";
      }

      return {
        transferResult: blockedResult,
        previewState: null,
        ownedByWallet: null,
        ownedByWalletError: null,
        transferError: caughtError.message,
        transferModalError: null,
        shouldOpenTransferModal: false,
        operationId,
        recovered: false,
      };
    }

    let ownershipResult: ScanOwnershipResult = {
      ownedByWallet: null,
      error: null,
    };

    if (
      normalizedAssetId &&
      isRetryableScanTransferError(
        caughtError,
        deps.isReturnInProgressOpenedError,
      )
    ) {
      ownershipResult = await deps.checkOwnershipByAssetId({
        assetId: normalizedAssetId,
        headers: input.headers,
        retryAfterTransfer: true,
      });

      if (ownershipResult.ownedByWallet === true) {
        const recovery =
          await recoverScanTransferAfterOwnershipConfirmed(
            deps,
            {
              productId: normalizedProductId,
              assetId: normalizedAssetId,
              operationId,
            },
          );

        if (
          recovery.recovered &&
          recovery.transferResult
        ) {
          return {
            transferResult: recovery.transferResult,
            previewState: recovery.previewState,
            ownedByWallet: true,
            ownedByWalletError: null,
            transferError: null,
            transferModalError: null,
            shouldOpenTransferModal: true,
            operationId: recovery.operationId,
            recovered: true,
          };
        }
      }
    }

    const message = toErrorMessage(caughtError);

    return {
      transferResult: null,
      previewState: null,
      ownedByWallet: ownershipResult.ownedByWallet,
      ownedByWalletError: ownershipResult.error,
      transferError: message,
      transferModalError: message,
      shouldOpenTransferModal: false,
      operationId,
      recovered: false,
    };
  }
}