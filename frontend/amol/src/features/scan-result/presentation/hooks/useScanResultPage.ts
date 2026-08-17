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
  loadPreviewState,
  resolveOwnedWalletTokenByAssetId,
  transferScanPurchased,
} from "../../infrastructure/scanResultApi";

import { getOptionalAuthHeaders } from "../../../../lib/authHeaders";

import type {
  MallScanTransferResponse,
  PreviewState,
  ScanResultPageState,
} from "../../../shared/types/scanResult";

import type { ProductBlueprintReviewPage } from "../../../shared/types/review";

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
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
    return createScanResultPageViewModel({
      previewState,
      ownedByWallet,
    });
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

  const runAutoTransferIfNeeded = useCallback(
    async (pid: string, headers?: HeadersInit) => {
      const normalizedProductId = pid.trim();

      if (!normalizedProductId) return;
      if (!headers) return;
      if (autoTransferTriggeredRef.current || transferringRef.current) return;

      autoTransferTriggeredRef.current = true;
      transferringRef.current = true;

      setBusyTransfer(true);
      setTransferError(null);
      setTransferModalError(null);

      try {
        if (!operationIdRef.current) {
          operationIdRef.current = globalThis.crypto.randomUUID();
        }

        const nextTransferResult = await transferScanPurchased({
          productId: normalizedProductId,
          operationId: operationIdRef.current,
          headers,
        });

        if (!mountedRef.current) return;

        setTransferResult(nextTransferResult);
        setTransferError(null);
        setTransferModalError(null);

        if (nextTransferResult.matched) {
          setTransferModalOpen(true);
        }
      } catch (caughtError) {
        if (!mountedRef.current) return;

        const message =
          caughtError instanceof Error
            ? caughtError.message
            : String(caughtError);

        setTransferResult(null);
        setTransferError(message);
        setTransferModalError(message);
        setTransferModalOpen(false);
      } finally {
        transferringRef.current = false;

        if (mountedRef.current) {
          setBusyTransfer(false);
        }
      }
    },
    [],
  );

  const loadAuthFlow = useCallback(
    async (pid: string) => {
      const normalizedProductId = pid.trim();
      if (!normalizedProductId) return;

      const headers = await getOptionalAuthHeaders();
      const hasAuth = Boolean(headers);

      if (mountedRef.current) {
        setAuthAvailable(hasAuth);
      }

      if (!headers) return;

      await runAutoTransferIfNeeded(normalizedProductId, headers);
    },
    [runAutoTransferIfNeeded],
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
    operationIdRef.current = globalThis.crypto.randomUUID();

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
      await loadAuthFlow(pid);
    } catch (caughtError) {
      if (!mountedRef.current) return;

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

      if (busyReviews) return;

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

        if (!mountedRef.current) return;

        setReviews(response);
        setReviewsError(null);
        setReviewPage(nextPage);
      } catch (caughtError) {
        if (!mountedRef.current) return;

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
    if (!previewAssetId) {
      setOwnedByWallet(null);
      setOwnedByWalletError(null);
      return;
    }

    if (busyOwnedByWallet) return;

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

      const owned = await isOwnedByWalletAssetId(
        previewAssetId,
        headers,
      );

      if (!mountedRef.current) return;

      setOwnedByWallet(owned);
      setOwnedByWalletError(null);
    } catch (caughtError) {
      if (!mountedRef.current) return;

      setOwnedByWallet(null);
      setOwnedByWalletError(
        caughtError instanceof Error
          ? caughtError.message
          : String(caughtError),
      );
    } finally {
      if (mountedRef.current) {
        setBusyOwnedByWallet(false);
      }
    }
  }, [busyOwnedByWallet, previewAssetId]);

  const openContentsAfterResolve = useCallback(() => {
    if (!transferSuccessModalViewModel?.canOpenContents) return;

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
      if (!assetId) return;

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
          searchParams.set("tokenBlueprintId", token.tokenBlueprintId);
        }

        if (tokenBlueprintPatch?.tokenName) {
          searchParams.set("tokenName", tokenBlueprintPatch.tokenName);
        }

        if (tokenBlueprintPatch?.tokenIcon) {
          searchParams.set("tokenIconUrl", tokenBlueprintPatch.tokenIcon);
        }

        navigate(`/contents?${searchParams.toString()}`);
      } catch (caughtError) {
        if (!mountedRef.current) return;

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

      if (postingReview) return false;

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
    if (busyReviews) return;
    if (reviews?.hasNext !== true) return;

    await loadReviews(reviewPage + 1);
  }, [
    busyReviews,
    loadReviews,
    reviewPage,
    reviews?.hasNext,
  ]);

  const prevReviewsPage = useCallback(async () => {
    if (busyReviews) return;
    if (reviewPage <= 1) return;

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
    if (!productBlueprintId) return;

    void loadReviews(1);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [productBlueprintId]);

  useEffect(() => {
    if (!previewAssetId) return;

    void loadOwnedState();

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [previewAssetId]);

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