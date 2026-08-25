// frontend/amol/src/features/scan-result/presentation/hooks/useScanResultPage.ts

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import {
  createScanResultPageViewModel,
  createScanTransferSuccessModalViewModel,
  loadScanReviews,
  submitScanReview,
  toScanReviewErrorMessage,
} from "../../application";
import {
  createProductBlueprintReview,
  fetchReviewsByProductBlueprintId,
  isOwnedByWalletAssetId,
  isReturnInProgressOpenedError,
  loadPreviewState,
  resolveOwnedWalletTokenByAssetId,
  transferScanPurchased,
} from "../../infrastructure/scanResultApi";
import { getOptionalAuthHeaders } from "../../../../lib/authHeaders";
import { HttpError } from "../../../../lib/http/httpError";
import type {
  MallScanTransferResponse,
  PreviewState,
  ScanResultPageState,
} from "../../../shared/types/scanResult";
import type { ProductBlueprintReviewPage } from "../../../shared/types/review";

const TRANSFER_OPERATION_STORAGE_PREFIX = "scan-transfer-operation:";
const OWNERSHIP_RETRY_ATTEMPTS = 6;
const TRANSFER_PREVIEW_RECOVERY_ATTEMPTS = 6;

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function wait(ms: number): Promise<void> {
  return new Promise((resolve) => {
    globalThis.setTimeout(resolve, ms);
  });
}

function getTransferOperationStorageKey(productId: string): string {
  return `${TRANSFER_OPERATION_STORAGE_PREFIX}${productId}`;
}

function readStoredTransferOperationId(productId: string): string {
  if (!productId) return "";
  try {
    return globalThis.sessionStorage.getItem(getTransferOperationStorageKey(productId))?.trim() ?? "";
  } catch {
    return "";
  }
}

function getOrCreateTransferOperationId(productId: string): string {
  const storedOperationId = readStoredTransferOperationId(productId);
  if (storedOperationId) {
    return storedOperationId;
  }

  const operationId = globalThis.crypto.randomUUID();
  try {
    globalThis.sessionStorage.setItem(getTransferOperationStorageKey(productId), operationId);
  } catch {
    // sessionStorage が利用できない場合も、現在の画面内では同じ operationId を利用する。
  }

  return operationId;
}

function clearStoredTransferOperationId(productId: string, operationId: string): void {
  if (!productId || !operationId) return;
  try {
    const storageKey = getTransferOperationStorageKey(productId);
    if (globalThis.sessionStorage.getItem(storageKey) === operationId) {
      globalThis.sessionStorage.removeItem(storageKey);
    }
  } catch {
    // sessionStorage が利用できない場合は何もしない。
  }
}

function isRetryableTransferError(error: unknown): boolean {
  if (isReturnInProgressOpenedError(error)) {
    return false;
  }

  if (error instanceof HttpError) {
    return error.status === 408 || error.status >= 500;
  }

  return error instanceof TypeError;
}

function createRecoveredTransferResult(
  previewState: PreviewState,
  assetId: string,
): MallScanTransferResponse | null {
  const transfers = previewState.raw.transfers;
  const latestTransfer = transfers.length > 0 ? transfers[transfers.length - 1] : undefined;
  const fromDisplayName =
    latestTransfer?.fromAvatarName?.trim() ||
    latestTransfer?.fromBrandName?.trim() ||
    "";
  const toDisplayName =
    latestTransfer?.toAvatarName?.trim() ||
    latestTransfer?.toBrandName?.trim() ||
    previewState.raw.owner?.avatarName?.trim() ||
    previewState.raw.owner?.brandName?.trim() ||
    "";

  if (!fromDisplayName || !toDisplayName) {
    return null;
  }

  return {
    avatarId: previewState.raw.owner?.avatarId ?? "",
    productId: previewState.raw.productId,
    matched: true,
    txSignature: "",
    fromDisplayName,
    toDisplayName,
    updatedToAddress: true,
    assetId,
  };
}

export function useScanProductIdFromUrl(): string {
  const params = useParams();
  const [searchParams] = useSearchParams();

  return useMemo(() => {
    const fromQuery = searchParams.get("productId");
    if (fromQuery?.trim()) return fromQuery.trim();

    const fromParams = params.productId;
    if (fromParams?.trim()) return safeDecodeURIComponent(fromParams.trim());

    return "";
  }, [params.productId, searchParams]);
}

export function useScanResultPage() {
  const navigate = useNavigate();
  const productId = useScanProductIdFromUrl();

  const [previewState, setPreviewState] = useState<PreviewState | null>(null);
  const [transferResult, setTransferResult] = useState<MallScanTransferResponse | null>(null);
  const [reviews, setReviews] = useState<ProductBlueprintReviewPage | null>(null);
  const [reviewsError, setReviewsError] = useState<string | null>(null);
  const [reviewPage, setReviewPage] = useState(1);
  const [reviewPerPage] = useState(20);
  const [busyReviews, setBusyReviews] = useState(false);
  const [ownedByWallet, setOwnedByWallet] = useState<boolean | null>(null);
  const [ownedByWalletError, setOwnedByWalletError] = useState<string | null>(null);
  const [busyOwnedByWallet, setBusyOwnedByWallet] = useState(false);
  const [postingReview, setPostingReview] = useState(false);
  const [postReviewError, setPostReviewError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busyTransfer, setBusyTransfer] = useState(false);
  const [transferError, setTransferError] = useState<string | null>(null);
  const [authAvailable, setAuthAvailable] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [transferModalOpen, setTransferModalOpen] = useState(false);
  const [transferModalError, setTransferModalError] = useState<string | null>(null);

  const autoTransferTriggeredRef = useRef(false);
  const mountedRef = useRef(true);
  const loadingProductIdRef = useRef("");
  const transferringRef = useRef(false);
  const operationIdRef = useRef("");

  const productBlueprintId = previewState?.raw.productBlueprintId ?? "";
  const previewAssetId = previewState?.raw.token?.assetId ?? "";
  const transferredAssetId = transferResult?.assetId ?? "";
  const transferTxSignature = transferResult?.txSignature ?? "";
  const transferMatched = transferResult?.matched ?? false;
  const hasMultipleTransfers = (previewState?.raw.transfers.length ?? 0) >= 2;

  const state: ScanResultPageState = {
    productId,
    previewState,
    transferResult,
    transferredAssetId,
    transferTxSignature,
    transferMatched,
    reviews,
    reviewsError,
    reviewPage,
    reviewPerPage,
    busyReviews,
    ownedByWallet,
    ownedByWalletError,
    busyOwnedByWallet,
    postingReview,
    postReviewError,
    loading,
    error,
    authAvailable,
    busyTransfer,
    transferError,
  };

  const viewModel = useMemo(() => {
    return createScanResultPageViewModel({ previewState, ownedByWallet });
  }, [ownedByWallet, previewState]);

  const transferSuccessModalViewModel = useMemo(() => {
    return createScanTransferSuccessModalViewModel({
      result: transferResult,
      token: previewState?.raw.token ?? null,
      tokenBlueprintPatch: previewState?.raw.tokenBlueprintPatch ?? null,
      productName: previewState?.raw.productBlueprintPatch?.productName ?? "",
    });
  }, [
    previewState?.raw.productBlueprintPatch?.productName,
    previewState?.raw.token,
    previewState?.raw.tokenBlueprintPatch,
    transferResult,
  ]);

  const closeTransferModal = useCallback(() => {
    setTransferModalOpen(false);
    setTransferModalError(null);
  }, []);

  const checkOwnedStateByAssetId = useCallback(
    async (
      assetId: string,
      headers: HeadersInit,
      retryAfterTransfer = false,
    ): Promise<boolean | null> => {
      const normalizedAssetId = assetId.trim();

      if (!normalizedAssetId) {
        if (mountedRef.current) {
          setOwnedByWallet(null);
          setOwnedByWalletError(null);
        }

        return null;
      }

      setBusyOwnedByWallet(true);
      setOwnedByWalletError(null);

      const maxAttempts = retryAfterTransfer ? OWNERSHIP_RETRY_ATTEMPTS : 1;
      let lastError: unknown = null;

      try {
        for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
          try {
            const owned = await isOwnedByWalletAssetId(normalizedAssetId, headers);

            if (!mountedRef.current) {
              return null;
            }

            if (owned) {
              setOwnedByWallet(true);
              setOwnedByWalletError(null);
              return true;
            }

            if (!retryAfterTransfer || attempt >= maxAttempts) {
              setOwnedByWallet(false);
              setOwnedByWalletError(null);
              return false;
            }
          } catch (caughtError) {
            lastError = caughtError;

            if (attempt >= maxAttempts) {
              break;
            }
          }

          await wait(700 * attempt);

          if (!mountedRef.current) {
            return null;
          }
        }

        if (!mountedRef.current) {
          return null;
        }

        setOwnedByWallet(null);
        setOwnedByWalletError(
          lastError instanceof Error
            ? lastError.message
            : String(lastError),
        );

        return null;
      } finally {
        if (mountedRef.current) {
          setBusyOwnedByWallet(false);
        }
      }
    },
    [],
  );

  const recoverTransferAfterOwnershipConfirmed = useCallback(
    async (
      pid: string,
      assetId: string,
    ): Promise<MallScanTransferResponse | null> => {
      const normalizedProductId = pid.trim();
      const normalizedAssetId = assetId.trim();

      if (!normalizedProductId || !normalizedAssetId) {
        return null;
      }

      for (
        let attempt = 1;
        attempt <= TRANSFER_PREVIEW_RECOVERY_ATTEMPTS;
        attempt += 1
      ) {
        try {
          const refreshedPreviewState = await loadPreviewState(normalizedProductId);

          if (!mountedRef.current) {
            return null;
          }

          setPreviewState(refreshedPreviewState);

          const recoveredResult = createRecoveredTransferResult(
            refreshedPreviewState,
            normalizedAssetId,
          );

          if (recoveredResult) {
            setTransferResult(recoveredResult);
            setTransferError(null);
            setTransferModalError(null);
            setTransferModalOpen(true);
            setOwnedByWallet(true);
            setOwnedByWalletError(null);

            clearStoredTransferOperationId(
              normalizedProductId,
              operationIdRef.current,
            );
            operationIdRef.current = "";

            return recoveredResult;
          }
        } catch {
          // transfer 自体は完了している可能性があるため、preview BFF の反映を待って再試行する。
        }

        if (attempt < TRANSFER_PREVIEW_RECOVERY_ATTEMPTS) {
          await wait(700 * attempt);

          if (!mountedRef.current) {
            return null;
          }
        }
      }

      return null;
    },
    [],
  );

  const runAutoTransferIfNeeded = useCallback(
    async (
      pid: string,
      assetId: string,
      headers?: HeadersInit,
    ): Promise<MallScanTransferResponse | null> => {
      const normalizedProductId = pid.trim();
      const normalizedAssetId = assetId.trim();

      if (!normalizedProductId || !headers) {
        return null;
      }

      if (autoTransferTriggeredRef.current || transferringRef.current) {
        return null;
      }

      autoTransferTriggeredRef.current = true;
      transferringRef.current = true;
      setBusyTransfer(true);
      setTransferError(null);
      setTransferModalError(null);

      try {
        operationIdRef.current =
          operationIdRef.current ||
          getOrCreateTransferOperationId(normalizedProductId);

        const nextTransferResult = await transferScanPurchased({
          productId: normalizedProductId,
          operationId: operationIdRef.current,
          headers,
        });

        if (!mountedRef.current) {
          return nextTransferResult;
        }

        setTransferResult(nextTransferResult);
        setTransferError(null);
        setTransferModalError(null);

        clearStoredTransferOperationId(
          normalizedProductId,
          operationIdRef.current,
        );
        operationIdRef.current = "";

        if (nextTransferResult.matched) {
          setTransferModalOpen(true);
        }

        return nextTransferResult;
      } catch (caughtError) {
        if (!mountedRef.current) {
          return null;
        }

        if (isReturnInProgressOpenedError(caughtError)) {
          const blockedResult: MallScanTransferResponse = {
            avatarId: caughtError.avatarId,
            productId:
              caughtError.productId ||
              normalizedProductId,
            matched: false,
            matchedOrderId:
              caughtError.matchedOrderId,
            matchedItemIndex:
              caughtError.matchedItemIndex,
            txSignature: "",
            fromDisplayName: "",
            toDisplayName: "",
            updatedToAddress: false,
            assetId: normalizedAssetId,
          };

          setTransferResult(blockedResult);
          setTransferError(caughtError.message);
          setTransferModalError(null);
          setTransferModalOpen(false);

          clearStoredTransferOperationId(
            normalizedProductId,
            operationIdRef.current,
          );
          operationIdRef.current = "";

          return blockedResult;
        }

        if (isRetryableTransferError(caughtError) && normalizedAssetId) {
          const owned = await checkOwnedStateByAssetId(
            normalizedAssetId,
            headers,
            true,
          );

          if (!mountedRef.current) {
            return null;
          }

          if (owned === true) {
            const recoveredResult =
              await recoverTransferAfterOwnershipConfirmed(
                normalizedProductId,
                normalizedAssetId,
              );

            if (recoveredResult) {
              return recoveredResult;
            }
          }
        }

        const message =
          caughtError instanceof Error
            ? caughtError.message
            : String(caughtError);

        setTransferResult(null);
        setTransferError(message);
        setTransferModalError(message);
        setTransferModalOpen(false);

        return null;
      } finally {
        transferringRef.current = false;

        if (mountedRef.current) {
          setBusyTransfer(false);
        }
      }
    },
    [
      checkOwnedStateByAssetId,
      recoverTransferAfterOwnershipConfirmed,
    ],
  );

  const loadAuthFlow = useCallback(
    async (
      pid: string,
      assetId: string,
    ) => {
      const normalizedProductId = pid.trim();
      const normalizedAssetId = assetId.trim();

      if (!normalizedProductId) {
        return;
      }

      const headers = await getOptionalAuthHeaders();
      const hasAuth = Boolean(headers);

      if (mountedRef.current) {
        setAuthAvailable(hasAuth);
      }

      if (!headers) {
        return;
      }

      if (normalizedAssetId) {
        const hasPendingTransferOperation = Boolean(
          operationIdRef.current,
        );

        const alreadyOwned = await checkOwnedStateByAssetId(
          normalizedAssetId,
          headers,
          hasPendingTransferOperation,
        );

        if (!mountedRef.current) {
          return;
        }

        if (alreadyOwned === true) {
          if (hasPendingTransferOperation) {
            await recoverTransferAfterOwnershipConfirmed(
              normalizedProductId,
              normalizedAssetId,
            );
          }

          return;
        }
      }

      const nextTransferResult = await runAutoTransferIfNeeded(
        normalizedProductId,
        normalizedAssetId,
        headers,
      );

      if (!mountedRef.current) {
        return;
      }

      const ownedCheckAssetId =
        nextTransferResult?.assetId?.trim() ||
        normalizedAssetId;

      if (!ownedCheckAssetId) {
        setOwnedByWallet(null);
        setOwnedByWalletError(null);
        return;
      }

      if (nextTransferResult?.matched === true) {
        void checkOwnedStateByAssetId(
          ownedCheckAssetId,
          headers,
          true,
        );
      }
    },
    [
      checkOwnedStateByAssetId,
      recoverTransferAfterOwnershipConfirmed,
      runAutoTransferIfNeeded,
    ],
  );

  const load = useCallback(async () => {
    const pid = productId.trim();

    setLoading(true);
    setError(null);
    setPreviewState(null);
    setTransferResult(null);
    setTransferError(null);
    setTransferModalOpen(false);
    setTransferModalError(null);
    setReviews(null);
    setReviewsError(null);
    setOwnedByWallet(null);
    setOwnedByWalletError(null);
    setPostReviewError(null);
    setReviewPage(1);
    setAuthAvailable(false);

    autoTransferTriggeredRef.current = false;
    transferringRef.current = false;
    operationIdRef.current =
      pid
        ? readStoredTransferOperationId(pid)
        : "";

    try {
      if (!pid) {
        throw new Error("商品ID が無いため、プレビューを取得しません。");
      }

      loadingProductIdRef.current = pid;
      const nextState = await loadPreviewState(pid);

      if (
        !mountedRef.current ||
        loadingProductIdRef.current !== pid
      ) {
        return;
      }

      setPreviewState(nextState);

      const nextAssetId = nextState.raw.token?.assetId ?? "";
      await loadAuthFlow(pid, nextAssetId);
    } catch (caughtError) {
      if (!mountedRef.current) {
        return;
      }

      setError(
        caughtError instanceof Error
          ? caughtError.message
          : String(caughtError),
      );
    } finally {
      if (mountedRef.current) {
        setLoading(false);
      }
    }
  }, [loadAuthFlow, productId]);

  const loadReviews = useCallback(
    async (nextPage = reviewPage) => {
      const pbId = productBlueprintId.trim();

      if (!pbId) {
        setReviews(null);
        setReviewsError("productBlueprintId is empty");
        return;
      }

      if (busyReviews) {
        return;
      }

      setBusyReviews(true);
      setReviewsError(null);

      try {
        const response = await loadScanReviews(
          {
            fetchReviewsByProductBlueprintId,
            createProductBlueprintReview,
            getOptionalAuthHeaders,
          },
          {
            productBlueprintId: pbId,
            page: nextPage,
            perPage: reviewPerPage,
          },
        );

        if (!mountedRef.current) {
          return;
        }

        setReviews(response);
        setReviewsError(null);
        setReviewPage(nextPage);
      } catch (caughtError) {
        if (!mountedRef.current) {
          return;
        }

        setReviews(null);
        setReviewsError(
          caughtError instanceof Error
            ? caughtError.message
            : String(caughtError),
        );
      } finally {
        if (mountedRef.current) {
          setBusyReviews(false);
        }
      }
    },
    [
      busyReviews,
      productBlueprintId,
      reviewPage,
      reviewPerPage,
    ],
  );

  const loadOwnedState = useCallback(async () => {
    const assetId = previewAssetId.trim();

    if (!assetId) {
      setOwnedByWallet(null);
      setOwnedByWalletError(null);
      return;
    }

    if (busyOwnedByWallet) {
      return;
    }

    const headers = await getOptionalAuthHeaders();

    if (!headers) {
      if (mountedRef.current) {
        setOwnedByWallet(null);
        setOwnedByWalletError(null);
      }

      return;
    }

    await checkOwnedStateByAssetId(assetId, headers);
  }, [
    busyOwnedByWallet,
    checkOwnedStateByAssetId,
    previewAssetId,
  ]);

  const openContentsAfterResolve = useCallback(() => {
    if (!transferSuccessModalViewModel?.canOpenContents) {
      return;
    }

    const searchParams = new URLSearchParams({
      assetId: transferSuccessModalViewModel.assetId,
      productId: transferSuccessModalViewModel.productId,
      brandId: transferSuccessModalViewModel.brandId,
      brandName: transferSuccessModalViewModel.brandName,
      productName: transferSuccessModalViewModel.productName,
      metadataUri: transferSuccessModalViewModel.metadataUri,
      tokenBlueprintId: transferSuccessModalViewModel.tokenBlueprintId,
      tokenName: transferSuccessModalViewModel.tokenName,
      tokenIconUrl: transferSuccessModalViewModel.tokenIconUrl,
    });

    closeTransferModal();
    navigate(`/contents?${searchParams.toString()}`);
  }, [
    closeTransferModal,
    navigate,
    transferSuccessModalViewModel,
  ]);

  const openTokenContentsByAssetId = useCallback(
    async (assetId: string) => {
      if (!assetId) {
        return;
      }

      const headers = await getOptionalAuthHeaders();

      if (!headers) {
        navigate("/signin");
        return;
      }

      try {
        setOwnedByWalletError(null);

        const resolved = await resolveOwnedWalletTokenByAssetId(
          assetId,
          headers,
        );

        if (!resolved.metadataUri) {
          throw new Error("metadataUri is empty");
        }

        const token = previewState?.raw.token;
        const tokenBlueprintPatch = previewState?.raw.tokenBlueprintPatch;

        const searchParams = new URLSearchParams({
          assetId: resolved.assetId,
          metadataUri: resolved.metadataUri,
          productId: resolved.productId,
          brandId: resolved.brandId,
          brandName: resolved.brandName,
          productName: resolved.productName,
          productBlueprintId: resolved.productBlueprintId,
        });

        if (token?.tokenBlueprintId) {
          searchParams.set(
            "tokenBlueprintId",
            token.tokenBlueprintId,
          );
        }

        if (tokenBlueprintPatch?.tokenName) {
          searchParams.set(
            "tokenName",
            tokenBlueprintPatch.tokenName,
          );
        }

        if (tokenBlueprintPatch?.tokenIcon) {
          searchParams.set(
            "tokenIconUrl",
            tokenBlueprintPatch.tokenIcon,
          );
        }

        navigate(`/contents?${searchParams.toString()}`);
      } catch (caughtError) {
        if (!mountedRef.current) {
          return;
        }

        setOwnedByWalletError(
          caughtError instanceof Error
            ? caughtError.message
            : String(caughtError),
        );
      }
    },
    [navigate, previewState],
  );

  const submitReview = useCallback(
    async (body: string, rating: number) => {
      const pbId = productBlueprintId.trim();

      if (postingReview) {
        return false;
      }

      setPostingReview(true);
      setPostReviewError(null);

      try {
        await submitScanReview(
          {
            fetchReviewsByProductBlueprintId,
            createProductBlueprintReview,
            getOptionalAuthHeaders,
          },
          {
            productBlueprintId: pbId,
            body,
            rating,
          },
        );

        await loadReviews(1);

        setPostReviewError(null);
        return true;
      } catch (caughtError) {
        setPostReviewError(
          toScanReviewErrorMessage(caughtError),
        );

        return false;
      } finally {
        if (mountedRef.current) {
          setPostingReview(false);
        }
      }
    },
    [
      loadReviews,
      postingReview,
      productBlueprintId,
    ],
  );

  const nextReviewsPage = useCallback(async () => {
    if (busyReviews || reviews?.hasNext !== true) {
      return;
    }

    await loadReviews(reviewPage + 1);
  }, [
    busyReviews,
    loadReviews,
    reviewPage,
    reviews?.hasNext,
  ]);

  const prevReviewsPage = useCallback(async () => {
    if (busyReviews || reviewPage <= 1) {
      return;
    }

    await loadReviews(reviewPage - 1);
  }, [
    busyReviews,
    loadReviews,
    reviewPage,
  ]);

  useEffect(() => {
    mountedRef.current = true;
    void load();

    return () => {
      mountedRef.current = false;
    };

    // Intentionally depend only on productId.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [productId]);

  useEffect(() => {
    if (!productBlueprintId) {
      return;
    }

    void loadReviews(1);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [productBlueprintId]);

  return {
    state,
    viewModel,
    transferSuccessModalViewModel,
    hasMultipleTransfers,
    load,
    loadReviews,
    loadOwnedState,
    submitReview,
    nextReviewsPage,
    prevReviewsPage,
    openContentsAfterResolve,
    openTokenContentsByAssetId,
    transferModalOpen,
    transferModalError,
    closeTransferModal,
  };
}