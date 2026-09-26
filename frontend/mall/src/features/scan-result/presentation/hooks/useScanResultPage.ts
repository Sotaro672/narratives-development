// frontend/mall/src/features/scan-result/presentation/hooks/useScanResultPage.ts

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

import { getMyAvatar } from "../../../avatar/api/avatarApi";
import { getOptionalAuthHeaders } from "../../../../lib/authHeaders";

import { createScanResultPageViewModel } from "../../application/scanPageViewModelFactory";
import {
  checkScanOwnershipByAssetId,
} from "../../application/scanOwnershipUsecase";
import {
  loadScanReviews,
  submitScanReview,
  toScanReviewErrorMessage,
} from "../../application/scanReviewUsecase";
import { resolveScanResult } from "../../application/scanResolveUsecase";
import { executeScanTransfer } from "../../application/scanTransferUsecase";

import {
  createProductBlueprintReview,
  fetchReviewsByProductBlueprintId,
  isOwnedByWalletAssetId,
  isReturnInProgressOpenedError,
  loadPreviewState,
  transferScanPurchased,
} from "../../infrastructure/scanResultApi";
import {
  clearStoredTransferOperationId,
  getOrCreateTransferOperationId,
  readStoredTransferOperationId,
} from "../../infrastructure/scanTransferOperationStorage";

import type {
  MallScanTransferResponse,
  PreviewState,
  ScanResultPageState,
} from "../../../shared/types/scanResult";
import type { ProductBlueprintReviewPage } from "../../../shared/types/review";

import { useScanProductIdFromUrl } from "./useScanProductIdFromUrl";

export function useScanResultPage() {
  const navigate = useNavigate();
  const productId = useScanProductIdFromUrl();

  const [previewState, setPreviewState] = useState<PreviewState | null>(null);
  const [transferResult, setTransferResult] =
    useState<MallScanTransferResponse | null>(null);
  const [reviews, setReviews] =
    useState<ProductBlueprintReviewPage | null>(null);
  const [reviewsError, setReviewsError] = useState<string | null>(null);
  const [reviewPage, setReviewPage] = useState(1);
  const [reviewPerPage] = useState(20);
  const [busyReviews, setBusyReviews] = useState(false);
  const [ownedByWallet, setOwnedByWallet] = useState<boolean | null>(null);
  const [ownedByWalletError, setOwnedByWalletError] =
    useState<string | null>(null);
  const [busyOwnedByWallet, setBusyOwnedByWallet] = useState(false);
  const [currentAvatarId, setCurrentAvatarId] = useState("");
  const [postingReview, setPostingReview] = useState(false);
  const [postReviewError, setPostReviewError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busyTransfer, setBusyTransfer] = useState(false);
  const [transferError, setTransferError] = useState<string | null>(null);
  const [authAvailable, setAuthAvailable] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [transferConfirmationRequired, setTransferConfirmationRequired] =
    useState(false);
  const [transferConfirmModalOpen, setTransferConfirmModalOpen] =
    useState(false);
  const [transferModalOpen, setTransferModalOpen] = useState(false);
  const [transferModalError, setTransferModalError] =
    useState<string | null>(null);

  const mountedRef = useRef(true);
  const loadingProductIdRef = useRef("");

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
    transferConfirmationRequired,
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
    return createScanResultPageViewModel({
      previewState,
      ownedByWallet,
    });
  }, [ownedByWallet, previewState]);

  const closeTransferModal = useCallback(() => {
    setTransferModalOpen(false);
    setTransferModalError(null);
  }, []);

  const closeTransferConfirmModal = useCallback(() => {
    if (busyTransfer) {
      return;
    }

    setTransferConfirmModalOpen(false);
    setTransferModalError(null);
  }, [busyTransfer]);

  const load = useCallback(async () => {
    const pid = productId.trim();

    loadingProductIdRef.current = pid;

    setLoading(true);
    setBusyOwnedByWallet(true);
    setBusyTransfer(true);
    setError(null);
    setPreviewState(null);
    setTransferResult(null);
    setTransferError(null);
    setTransferConfirmationRequired(false);
    setTransferConfirmModalOpen(false);
    setTransferModalOpen(false);
    setTransferModalError(null);
    setReviews(null);
    setReviewsError(null);
    setOwnedByWallet(null);
    setOwnedByWalletError(null);
    setCurrentAvatarId("");
    setPostReviewError(null);
    setReviewPage(1);
    setAuthAvailable(false);

    try {
      const result = await resolveScanResult(
        {
          loadPreviewState,
          getOptionalAuthHeaders,
          getMyAvatar,
          isOwnedByWalletAssetId,
          transferScanPurchased,
          isReturnInProgressOpenedError,
          readStoredTransferOperationId,
          getOrCreateTransferOperationId,
          clearStoredTransferOperationId,
        },
        {
          productId: pid,
        },
      );

      if (
        !mountedRef.current ||
        loadingProductIdRef.current !== pid
      ) {
        return;
      }

      setPreviewState(result.previewState);
      setCurrentAvatarId(result.currentAvatarId);
      setAuthAvailable(result.authAvailable);
      setOwnedByWallet(result.ownedByWallet);
      setOwnedByWalletError(result.ownedByWalletError);
      setTransferResult(result.transferResult);
      setTransferError(result.transferError);
      setTransferModalError(result.transferModalError);
      setTransferConfirmationRequired(
        result.requiresTransferConfirmation,
      );
      setTransferConfirmModalOpen(
        result.requiresTransferConfirmation,
      );
      setTransferModalOpen(result.shouldOpenTransferModal);
    } catch (caughtError) {
      if (
        !mountedRef.current ||
        loadingProductIdRef.current !== pid
      ) {
        return;
      }

      setError(
        caughtError instanceof Error
          ? caughtError.message
          : String(caughtError),
      );
    } finally {
      if (
        mountedRef.current &&
        loadingProductIdRef.current === pid
      ) {
        setLoading(false);
        setBusyOwnedByWallet(false);
        setBusyTransfer(false);
      }
    }
  }, [productId]);

  const confirmTransfer = useCallback(async () => {
    const pid = productId.trim();
    const assetId = previewAssetId.trim();

    if (busyTransfer || !transferConfirmationRequired) {
      return;
    }

    if (!pid || !assetId || !previewState) {
      setTransferModalError(
        "トークン移譲に必要な情報を取得できませんでした。",
      );
      return;
    }

    setBusyTransfer(true);
    setTransferError(null);
    setTransferModalError(null);
    setTransferModalOpen(false);

    try {
      const headers = await getOptionalAuthHeaders();

      if (
        !mountedRef.current ||
        loadingProductIdRef.current !== pid
      ) {
        return;
      }

      if (!headers) {
        setTransferConfirmationRequired(false);
        setTransferConfirmModalOpen(false);
        navigate("/signin");
        return;
      }

      const transfer = await executeScanTransfer(
        {
          transferScanPurchased,
          loadPreviewState,
          checkOwnershipByAssetId: (input) =>
            checkScanOwnershipByAssetId(
              {
                isOwnedByWalletAssetId,
              },
              input,
            ),
          isReturnInProgressOpenedError,
          getOrCreateTransferOperationId,
          clearStoredTransferOperationId,
        },
        {
          productId: pid,
          assetId,
          headers,
          operationId:
            readStoredTransferOperationId(pid).trim(),
        },
      );

      if (
        !mountedRef.current ||
        loadingProductIdRef.current !== pid
      ) {
        return;
      }

      const resolvedPreviewState =
        transfer.previewState ?? previewState;

      if (transfer.previewState) {
        setPreviewState(transfer.previewState);
      }

      let nextOwnedByWallet =
        transfer.ownedByWallet ?? ownedByWallet;
      let nextOwnedByWalletError =
        transfer.ownedByWalletError ?? ownedByWalletError;

      const ownedCheckAssetId =
        transfer.transferResult?.assetId?.trim() ||
        resolvedPreviewState.raw.token?.assetId?.trim() ||
        assetId;

      if (
        transfer.transferResult?.matched === true &&
        ownedCheckAssetId &&
        transfer.ownedByWallet !== true
      ) {
        const ownershipAfterTransfer =
          await checkScanOwnershipByAssetId(
            {
              isOwnedByWalletAssetId,
            },
            {
              assetId: ownedCheckAssetId,
              headers,
              retryAfterTransfer: true,
            },
          );

        if (
          !mountedRef.current ||
          loadingProductIdRef.current !== pid
        ) {
          return;
        }

        nextOwnedByWallet =
          ownershipAfterTransfer.ownedByWallet;
        nextOwnedByWalletError =
          ownershipAfterTransfer.error;
      }

      setTransferResult(transfer.transferResult);
      setOwnedByWallet(nextOwnedByWallet);
      setOwnedByWalletError(nextOwnedByWalletError);
      setTransferError(transfer.transferError);
      setTransferModalError(transfer.transferModalError);

      if (transfer.shouldOpenTransferModal) {
        setTransferConfirmationRequired(false);
        setTransferConfirmModalOpen(false);

        const matchedOrderId =
          transfer.transferResult?.matchedOrderId?.trim() ?? "";
        const matchedItemIndex =
          transfer.transferResult?.matchedItemIndex;

        if (
          transfer.transferResult?.matchedItemType === "resale" &&
          matchedOrderId &&
          typeof matchedItemIndex === "number" &&
          Number.isInteger(matchedItemIndex) &&
          matchedItemIndex >= 0
        ) {
          navigate(
            `/avatar-reviews/order-items/${encodeURIComponent(
              matchedOrderId,
            )}/${matchedItemIndex}/new`,
          );
          return;
        }

        setTransferModalOpen(true);
        return;
      }

      if (transfer.transferResult) {
        setTransferConfirmationRequired(false);
        setTransferConfirmModalOpen(false);
      }
    } catch (caughtError) {
      if (
        !mountedRef.current ||
        loadingProductIdRef.current !== pid
      ) {
        return;
      }

      const message =
        caughtError instanceof Error
          ? caughtError.message
          : String(caughtError);

      setTransferError(message);
      setTransferModalError(message);
    } finally {
      if (
        mountedRef.current &&
        loadingProductIdRef.current === pid
      ) {
        setBusyTransfer(false);
      }
    }
  }, [
    busyTransfer,
    navigate,
    ownedByWallet,
    ownedByWalletError,
    previewAssetId,
    previewState,
    productId,
    transferConfirmationRequired,
  ]);

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

    setBusyOwnedByWallet(true);
    setOwnedByWalletError(null);

    try {
      const headers = await getOptionalAuthHeaders();

      if (!headers) {
        if (mountedRef.current) {
          setOwnedByWallet(null);
          setOwnedByWalletError(null);
        }

        return;
      }

      const result = await checkScanOwnershipByAssetId(
        {
          isOwnedByWalletAssetId,
        },
        {
          assetId,
          headers,
        },
      );

      if (!mountedRef.current) {
        return;
      }

      setOwnedByWallet(result.ownedByWallet);
      setOwnedByWalletError(result.error);
    } finally {
      if (mountedRef.current) {
        setBusyOwnedByWallet(false);
      }
    }
  }, [
    busyOwnedByWallet,
    previewAssetId,
  ]);

  const openTokenContentsByAssetId = useCallback(
    async (assetId: string) => {
      const normalizedAssetId = assetId.trim();

      if (!normalizedAssetId || !previewState) {
        return;
      }

      const headers = await getOptionalAuthHeaders();

      if (!headers) {
        navigate("/signin");
        return;
      }

      const preview = previewState.raw;
      const token = preview.token;
      const productBlueprintPatch = preview.productBlueprintPatch;
      const tokenBlueprintPatch = preview.tokenBlueprintPatch;

      const metadataUri = token?.metadataUri?.trim() ?? "";
      const tokenBlueprintId = token?.tokenBlueprintId?.trim() ?? "";

      if (!metadataUri || !tokenBlueprintId) {
        setOwnedByWalletError(
          "トークンコンテンツ情報を取得できませんでした。",
        );
        return;
      }

      setOwnedByWalletError(null);

      const searchParams = new URLSearchParams({
        assetId: normalizedAssetId,
        metadataUri,
        productId: preview.productId,
        brandId:
          token?.brandId?.trim() ||
          productBlueprintPatch?.brandId?.trim() ||
          "",
        brandName:
          token?.brandName?.trim() ||
          preview.brandName?.trim() ||
          tokenBlueprintPatch?.brandName?.trim() ||
          "",
        productName:
          productBlueprintPatch?.productName?.trim() || "",
        productBlueprintId: preview.productBlueprintId,
        tokenBlueprintId,
      });

      navigate(`/contents?${searchParams.toString()}`);
    },
    [navigate, previewState],
  );

  const openContentsAfterResolve = useCallback(async () => {
    const assetId = transferredAssetId.trim();

    if (!assetId || !transferMatched) {
      return;
    }

    closeTransferModal();
    await openTokenContentsByAssetId(assetId);
  }, [
    closeTransferModal,
    openTokenContentsByAssetId,
    transferMatched,
    transferredAssetId,
  ]);

  const submitReview = useCallback(
    async (
      body: string,
      rating: number,
    ) => {
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
    if (
      busyReviews ||
      reviews?.hasNext !== true
    ) {
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
    if (
      busyReviews ||
      reviewPage <= 1
    ) {
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

    // productId が変化した場合のみScanResult全体を再解決する。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [productId]);

  useEffect(() => {
    if (!productBlueprintId) {
      return;
    }

    void loadReviews(1);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [productBlueprintId]);

  const canOpenTransferContents =
    transferMatched &&
    Boolean(transferredAssetId.trim());

  return {
    state,
    viewModel,
    currentAvatarId,
    hasMultipleTransfers,
    canOpenTransferContents,
    load,
    loadReviews,
    loadOwnedState,
    submitReview,
    nextReviewsPage,
    prevReviewsPage,
    openContentsAfterResolve,
    openTokenContentsByAssetId,
    transferConfirmModalOpen,
    transferModalOpen,
    transferModalError,
    confirmTransfer,
    closeTransferConfirmModal,
    closeTransferModal,
  };
}