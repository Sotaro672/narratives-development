// frontend/mall/src/features/scan-result/application/scanOwnershipUsecase.ts

import type { PreviewState } from "../../shared/types/scanResult";

export const OWNERSHIP_RETRY_ATTEMPTS = 6;
const OWNERSHIP_RETRY_BASE_DELAY_MS = 700;

export type ScanOwnershipUsecaseDeps = {
  isOwnedByWalletAssetId: (
    assetId: string,
    headers?: HeadersInit,
  ) => Promise<boolean>;
  wait?: (ms: number) => Promise<void>;
};

export type CheckScanOwnershipByAssetIdInput = {
  assetId: string;
  headers: HeadersInit;
  retryAfterTransfer?: boolean;
};

export type ResolveScanOwnershipInput = {
  previewState: PreviewState;
  currentAvatarId: string;
  headers: HeadersInit;
  retryAfterTransfer?: boolean;
};

export type ScanOwnershipResult = {
  ownedByWallet: boolean | null;
  error: string | null;
};

function defaultWait(ms: number): Promise<void> {
  return new Promise((resolve) => {
    globalThis.setTimeout(resolve, ms);
  });
}

function toErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function resolveOwnedByCurrentAvatar(
  previewState: PreviewState,
  currentAvatarId: string,
): boolean | null {
  const normalizedCurrentAvatarId = currentAvatarId.trim();
  const owner = previewState.raw.owner;

  if (!normalizedCurrentAvatarId || !owner) {
    return null;
  }

  if (owner.ownerType !== "avatar") {
    return false;
  }

  const ownerAvatarId = owner.avatarId?.trim() ?? "";

  if (!ownerAvatarId) {
    return null;
  }

  return ownerAvatarId === normalizedCurrentAvatarId;
}

export async function checkScanOwnershipByAssetId(
  deps: ScanOwnershipUsecaseDeps,
  input: CheckScanOwnershipByAssetIdInput,
): Promise<ScanOwnershipResult> {
  const normalizedAssetId = input.assetId.trim();

  if (!normalizedAssetId) {
    return {
      ownedByWallet: null,
      error: null,
    };
  }

  const retryAfterTransfer = input.retryAfterTransfer === true;
  const maxAttempts = retryAfterTransfer ? OWNERSHIP_RETRY_ATTEMPTS : 1;
  const wait = deps.wait ?? defaultWait;
  let lastError: unknown = null;

  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      const owned = await deps.isOwnedByWalletAssetId(
        normalizedAssetId,
        input.headers,
      );

      if (owned) {
        return {
          ownedByWallet: true,
          error: null,
        };
      }

      if (!retryAfterTransfer || attempt >= maxAttempts) {
        return {
          ownedByWallet: false,
          error: null,
        };
      }
    } catch (caughtError) {
      lastError = caughtError;

      if (attempt >= maxAttempts) {
        break;
      }
    }

    await wait(OWNERSHIP_RETRY_BASE_DELAY_MS * attempt);
  }

  return {
    ownedByWallet: null,
    error: lastError ? toErrorMessage(lastError) : null,
  };
}

export async function resolveScanOwnership(
  deps: ScanOwnershipUsecaseDeps,
  input: ResolveScanOwnershipInput,
): Promise<ScanOwnershipResult> {
  const ownedByCurrentAvatar = resolveOwnedByCurrentAvatar(
    input.previewState,
    input.currentAvatarId,
  );

  if (ownedByCurrentAvatar === true) {
    return {
      ownedByWallet: true,
      error: null,
    };
  }

  const assetId = input.previewState.raw.token?.assetId?.trim() ?? "";

  if (!assetId) {
    return {
      ownedByWallet: ownedByCurrentAvatar,
      error: null,
    };
  }

  return checkScanOwnershipByAssetId(deps, {
    assetId,
    headers: input.headers,
    retryAfterTransfer: input.retryAfterTransfer,
  });
}