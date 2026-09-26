// frontend/mall/src/features/scan-result/application/scanResolveUsecase.ts

import type {
  MallScanTransferResponse,
  PreviewState,
} from "../../shared/types/scanResult";

import {
  checkScanOwnershipByAssetId,
  resolveScanOwnership,
  type ScanOwnershipUsecaseDeps,
} from "./scanOwnershipUsecase";
import {
  recoverScanTransferAfterOwnershipConfirmed,
  type ReturnInProgressOpenedErrorLike,
  type ScanTransferUsecaseDeps,
} from "./scanTransferUsecase";

export type ScanResolveUsecaseDeps = {
  loadPreviewState: (productId: string) => Promise<PreviewState>;
  getOptionalAuthHeaders: () => Promise<HeadersInit | undefined>;
  getMyAvatar: () => Promise<{ avatarId: string } | null>;
  isOwnedByWalletAssetId: (
    assetId: string,
    headers?: HeadersInit,
  ) => Promise<boolean>;
  transferScanPurchased: (input: {
    productId: string;
    operationId: string;
    headers?: HeadersInit;
  }) => Promise<MallScanTransferResponse>;
  isReturnInProgressOpenedError: (
    error: unknown,
  ) => error is ReturnInProgressOpenedErrorLike;
  readStoredTransferOperationId: (productId: string) => string;
  getOrCreateTransferOperationId: (productId: string) => string;
  clearStoredTransferOperationId: (
    productId: string,
    operationId: string,
  ) => void;
  wait?: (ms: number) => Promise<void>;
};

export type ResolveScanResultInput = {
  productId: string;
};

export type ResolveScanResultResult = {
  previewState: PreviewState;
  currentAvatarId: string;
  authAvailable: boolean;
  ownedByWallet: boolean | null;
  ownedByWalletError: string | null;
  transferResult: MallScanTransferResponse | null;
  transferError: string | null;
  transferModalError: string | null;
  shouldOpenTransferModal: boolean;
  requiresTransferConfirmation: boolean;
  operationId: string;
};

function createOwnershipDeps(
  deps: ScanResolveUsecaseDeps,
): ScanOwnershipUsecaseDeps {
  return {
    isOwnedByWalletAssetId: deps.isOwnedByWalletAssetId,
    wait: deps.wait,
  };
}

function createTransferDeps(
  deps: ScanResolveUsecaseDeps,
  ownershipDeps: ScanOwnershipUsecaseDeps,
): ScanTransferUsecaseDeps {
  return {
    transferScanPurchased: deps.transferScanPurchased,
    loadPreviewState: deps.loadPreviewState,
    checkOwnershipByAssetId: (input) =>
      checkScanOwnershipByAssetId(ownershipDeps, input),
    isReturnInProgressOpenedError: deps.isReturnInProgressOpenedError,
    getOrCreateTransferOperationId: deps.getOrCreateTransferOperationId,
    clearStoredTransferOperationId: deps.clearStoredTransferOperationId,
    wait: deps.wait,
  };
}

async function resolveCurrentAvatarId(
  deps: ScanResolveUsecaseDeps,
): Promise<string> {
  try {
    const avatar = await deps.getMyAvatar();
    return avatar?.avatarId?.trim() ?? "";
  } catch {
    return "";
  }
}

export async function resolveScanResult(
  deps: ScanResolveUsecaseDeps,
  input: ResolveScanResultInput,
): Promise<ResolveScanResultResult> {
  const productId = input.productId.trim();

  if (!productId) {
    throw new Error(
      "商品ID が無いため、プレビューを取得しません。",
    );
  }

  const previewState = await deps.loadPreviewState(productId);
  const headers = await deps.getOptionalAuthHeaders();
  const authAvailable = Boolean(headers);

  if (!headers) {
    return {
      previewState,
      currentAvatarId: "",
      authAvailable: false,
      ownedByWallet: null,
      ownedByWalletError: null,
      transferResult: null,
      transferError: null,
      transferModalError: null,
      shouldOpenTransferModal: false,
      requiresTransferConfirmation: false,
      operationId: "",
    };
  }

  const currentAvatarId = await resolveCurrentAvatarId(deps);
  const assetId = previewState.raw.token?.assetId?.trim() ?? "";
  const storedOperationId =
    deps.readStoredTransferOperationId(productId).trim();
  const hasPendingTransferOperation = Boolean(storedOperationId);

  if (!assetId) {
    return {
      previewState,
      currentAvatarId,
      authAvailable,
      ownedByWallet: null,
      ownedByWalletError: null,
      transferResult: null,
      transferError: null,
      transferModalError: null,
      shouldOpenTransferModal: false,
      requiresTransferConfirmation: false,
      operationId: storedOperationId,
    };
  }

  const ownershipDeps = createOwnershipDeps(deps);
  const transferDeps = createTransferDeps(
    deps,
    ownershipDeps,
  );

  const initialOwnership = await resolveScanOwnership(
    ownershipDeps,
    {
      previewState,
      currentAvatarId,
      headers,
      retryAfterTransfer: hasPendingTransferOperation,
    },
  );

  if (initialOwnership.ownedByWallet === true) {
    if (hasPendingTransferOperation) {
      const recovery =
        await recoverScanTransferAfterOwnershipConfirmed(
          transferDeps,
          {
            productId,
            assetId,
            operationId: storedOperationId,
          },
        );

      if (
        recovery.recovered &&
        recovery.previewState &&
        recovery.transferResult
      ) {
        return {
          previewState: recovery.previewState,
          currentAvatarId,
          authAvailable,
          ownedByWallet: true,
          ownedByWalletError: null,
          transferResult: recovery.transferResult,
          transferError: null,
          transferModalError: null,
          shouldOpenTransferModal: true,
          requiresTransferConfirmation: false,
          operationId: recovery.operationId,
        };
      }

      return {
        previewState,
        currentAvatarId,
        authAvailable,
        ownedByWallet: true,
        ownedByWalletError: null,
        transferResult: null,
        transferError: null,
        transferModalError: null,
        shouldOpenTransferModal: false,
        requiresTransferConfirmation: false,
        operationId: recovery.operationId,
      };
    }

    return {
      previewState,
      currentAvatarId,
      authAvailable,
      ownedByWallet: true,
      ownedByWalletError: null,
      transferResult: null,
      transferError: null,
      transferModalError: null,
      shouldOpenTransferModal: false,
      requiresTransferConfirmation: false,
      operationId: storedOperationId,
    };
  }

  if (initialOwnership.ownedByWallet === false) {
    return {
      previewState,
      currentAvatarId,
      authAvailable,
      ownedByWallet: false,
      ownedByWalletError: initialOwnership.error,
      transferResult: null,
      transferError: null,
      transferModalError: null,
      shouldOpenTransferModal: false,
      requiresTransferConfirmation: true,
      operationId: storedOperationId,
    };
  }

  return {
    previewState,
    currentAvatarId,
    authAvailable,
    ownedByWallet: null,
    ownedByWalletError: initialOwnership.error,
    transferResult: null,
    transferError: null,
    transferModalError: null,
    shouldOpenTransferModal: false,
    requiresTransferConfirmation: false,
    operationId: storedOperationId,
  };
}