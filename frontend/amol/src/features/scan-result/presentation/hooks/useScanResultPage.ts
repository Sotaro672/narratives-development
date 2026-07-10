// frontend/amol/src/features/scan-result/hooks/useScanResultPage.ts
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import {
  createScanResultPageViewModel,
  createScanTransferSuccessModalViewModel,
  loadScanReviews,
  resolveScanOwnedWalletState,
  resolveTransferredTokenWithRetry,
  runScanAutoTransfer,
  submitScanReview,
  toScanReviewErrorMessage,
} from "../../application";
import {
  createProductBlueprintReview,
  fetchMeAvatar,
  fetchMeWallet,
  fetchReviewsByProductBlueprintId,
  getAuthHeadersOrUndefined,
  isOwnedByWalletMintAddress,
  loadPreviewState,
  resolveOwnedWalletTokenByMintAddress,
  resolveTokenByMintAddress,
  transferScanPurchased,
} from "../../infrastructure/scanResultApi";
import type {
  CatalogReviewPage,
  MallOwnerInfo,
  MallScanTransferResponse,
  MallScanVerifyResponse,
  PreviewState,
  ScanResultPageState,
  TokenResolveDTO,
} from "../../types";

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function hasAuthorization(headers?: HeadersInit): boolean {
  if (!headers) {
    return false;
  }

  const h = new Headers(headers);

  return Boolean((h.get("Authorization") || h.get("authorization") || "").trim());
}

function wait(ms: number): Promise<void> {
  return new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

function verifyResultFromTransferResult(
  result: MallScanTransferResponse,
): MallScanVerifyResponse {
  return {
    avatarId: result.avatarId,
    productId: result.productId,
    scannedModelId: "",
    scannedTokenBlueprintId: "",
    purchasedPairs: [],
    matched: result.matched,
    match: null,
  };
}

export function useScanProductIdFromUrl(): string {
  const params = useParams();
  const [searchParams] = useSearchParams();

  return useMemo(() => {
    const fromQuery = searchParams.get("productId");

    if (fromQuery?.trim()) {
      return fromQuery.trim();
    }

    const fromParams = params.productId;

    if (fromParams?.trim()) {
      return safeDecodeURIComponent(fromParams.trim());
    }

    return "";
  }, [params.productId, searchParams]);
}

export function useScanResultPage() {
  const navigate = useNavigate();
  const productId = useScanProductIdFromUrl();

  const [previewState, setPreviewState] = useState<PreviewState | null>(null);
  const [meAvatar, setMeAvatar] = useState<MallOwnerInfo | null>(null);
  const [verifyResult, setVerifyResult] =
    useState<MallScanVerifyResponse | null>(null);
  const [transferResult, setTransferResult] =
    useState<MallScanTransferResponse | null>(null);

  const [reviews, setReviews] = useState<CatalogReviewPage | null>(null);
  const [reviewsError, setReviewsError] = useState<string | null>(null);
  const [reviewPage, setReviewPage] = useState(1);
  const [reviewPerPage] = useState(20);
  const [busyReviews, setBusyReviews] = useState(false);

  const [ownedByWallet, setOwnedByWallet] = useState<boolean | null>(null);
  const [ownedByWalletError, setOwnedByWalletError] = useState<string | null>(
    null,
  );
  const [busyOwnedByWallet, setBusyOwnedByWallet] = useState(false);

  const [postingReview, setPostingReview] = useState(false);
  const [postReviewError, setPostReviewError] = useState<string | null>(null);

  const [transferredMintOverride, setTransferredMintOverride] = useState("");

  const [resolvingTransferredToken, setResolvingTransferredToken] =
    useState(false);
  const [resolvedTransferredToken, setResolvedTransferredToken] =
    useState<TokenResolveDTO | null>(null);

  const [loading, setLoading] = useState(true);
  const [busyTransfer, setBusyTransfer] = useState(false);
  const [authAvailable, setAuthAvailable] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [transferModalOpen, setTransferModalOpen] = useState(false);
  const [transferModalError, setTransferModalError] = useState<string | null>(
    null,
  );

  const autoTransferTriggeredRef = useRef(false);
  const mountedRef = useRef(true);
  const loadingProductIdRef = useRef("");
  const transferringRef = useRef(false);

  const productBlueprintId = previewState?.raw.productBlueprintId.trim() || "";
  const previewMintAddress = previewState?.raw.token?.mintAddress.trim() || "";
  const transferredMintAddress =
    transferResult?.mintAddress.trim() || transferredMintOverride.trim();
  const transferTxSignature = transferResult?.txSignature.trim() || "";
  const transferMatched = transferResult?.matched ?? false;

  const displayTransfers = useMemo(
    () => previewState?.raw.transfers ?? [],
    [previewState?.raw.transfers],
  );

  const hasMultipleTransfers = displayTransfers.length >= 2;

  const state: ScanResultPageState = {
    productId,
    previewState,
    meAvatar,
    verifyResult,
    transferResult,
    transferredMintAddress,
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

    resolvingTransferredToken,
    resolvedTransferredToken,

    loading,
    error,
    authAvailable,
    busyTransfer,
  };

  const viewModel = useMemo(() => {
    return createScanResultPageViewModel({
      state,
      previewState,
      chainTransfers: displayTransfers,
    });
  }, [displayTransfers, previewState, state]);

  const transferSuccessModalViewModel = useMemo(() => {
    return createScanTransferSuccessModalViewModel({
      result: transferResult,
      transferredMintAddress,
      token: previewState?.raw.token ?? null,
      tokenBlueprintPatch: previewState?.tokenBlueprintPatch ?? null,
      productName: previewState?.raw.productBlueprintPatch?.productName ?? "",
    });
  }, [
    previewState?.raw.productBlueprintPatch?.productName,
    previewState?.raw.token,
    previewState?.tokenBlueprintPatch,
    transferResult,
    transferredMintAddress,
  ]);

  const closeTransferModal = useCallback(() => {
    setTransferModalOpen(false);
    setTransferModalError(null);
  }, []);

  const runAutoTransferIfNeeded = useCallback(
    async (pid: string, headers?: HeadersInit) => {
      const normalizedProductId = pid.trim();

      if (!normalizedProductId) {
        return;
      }

      if (!hasAuthorization(headers)) {
        return;
      }

      if (autoTransferTriggeredRef.current || transferringRef.current) {
        return;
      }

      autoTransferTriggeredRef.current = true;
      transferringRef.current = true;
      setBusyTransfer(true);

      try {
        const result = await runScanAutoTransfer(
          {
            fetchMeWallet,
            transferScanPurchased,
          },
          {
            productId: normalizedProductId,
            headers,
          },
        );

        if (result.transferredMintAddress) {
          setTransferredMintOverride(result.transferredMintAddress);
        }

        if (!mountedRef.current) {
          return;
        }

        setTransferResult(result.transferResult);
        setVerifyResult(verifyResultFromTransferResult(result.transferResult));

        if (result.transferResult.matched) {
          setTransferModalError(null);
          setTransferModalOpen(true);
        }
      } catch {
        // Auto transfer is best-effort.
        // Backend owns verification / ownership / purchase checks.
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

      if (!normalizedProductId) {
        return;
      }

      const headers = await getAuthHeadersOrUndefined();
      const hasAuth = hasAuthorization(headers);

      setAuthAvailable(hasAuth);

      if (!hasAuth) {
        return;
      }

      try {
        const avatar = await fetchMeAvatar(headers);

        if (!mountedRef.current) {
          return;
        }

        setMeAvatar(avatar);

        /**
         * Frontend verify is intentionally disabled.
         * transferScanPurchased is the single authoritative operation.
         * Backend handles purchase / ownership / eligibility checks.
         */
        await runAutoTransferIfNeeded(normalizedProductId, headers);
      } catch {
        // Auth-scoped scan flow is best-effort.
      }
    },
    [runAutoTransferIfNeeded],
  );

  const load = useCallback(async () => {
    const pid = productId.trim();

    setLoading(true);
    setError(null);
    setPreviewState(null);
    setMeAvatar(null);
    setVerifyResult(null);
    setTransferResult(null);
    setTransferredMintOverride("");
    setResolvedTransferredToken(null);
    setTransferModalOpen(false);
    setTransferModalError(null);
    setReviews(null);
    setReviewsError(null);
    setOwnedByWallet(null);
    setOwnedByWalletError(null);
    setPostReviewError(null);
    setReviewPage(1);
    autoTransferTriggeredRef.current = false;
    transferringRef.current = false;

    try {
      if (!pid) {
        throw new Error("商品ID が無いため、プレビューを取得しません。");
      }

      loadingProductIdRef.current = pid;

      const nextState = await loadPreviewState(pid);

      if (!mountedRef.current || loadingProductIdRef.current !== pid) {
        return;
      }

      setPreviewState(nextState);

      await loadAuthFlow(pid);
    } catch (e) {
      if (!mountedRef.current) {
        return;
      }

      setError(e instanceof Error ? e.message : String(e));
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
        const res = await loadScanReviews(
          {
            fetchReviewsByProductBlueprintId,
            createProductBlueprintReview,
            getAuthHeadersOrUndefined,
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

        setReviews(res);
        setReviewsError(null);
        setReviewPage(nextPage);
      } catch (e) {
        if (!mountedRef.current) {
          return;
        }

        setReviews(null);
        setReviewsError(e instanceof Error ? e.message : String(e));
      } finally {
        if (mountedRef.current) {
          setBusyReviews(false);
        }
      }
    },
    [busyReviews, productBlueprintId, reviewPage, reviewPerPage],
  );

  const loadOwnedState = useCallback(async () => {
    const mintAddress = previewMintAddress.trim();

    if (!mintAddress) {
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
      const owned = await resolveScanOwnedWalletState(
        {
          getAuthHeadersOrUndefined,
          isOwnedByWalletMintAddress,
          hasAuthorization,
        },
        mintAddress,
      );

      if (!mountedRef.current) {
        return;
      }

      setOwnedByWallet(owned);
      setOwnedByWalletError(null);
    } catch (e) {
      if (!mountedRef.current) {
        return;
      }

      setOwnedByWallet(null);
      setOwnedByWalletError(e instanceof Error ? e.message : String(e));
    } finally {
      if (mountedRef.current) {
        setBusyOwnedByWallet(false);
      }
    }
  }, [busyOwnedByWallet, previewMintAddress]);

  const resolveTransferredTokenWithoutSync = useCallback(async () => {
    if (resolvingTransferredToken) {
      throw new Error("resolve is already running");
    }

    const mintAddress = transferredMintAddress.trim();

    if (!mintAddress) {
      throw new Error("transferred mintAddress is empty");
    }

    setResolvingTransferredToken(true);

    try {
      const resolved = await resolveTransferredTokenWithRetry(
        {
          getAuthHeadersOrUndefined,
          resolveTokenByMintAddress,
          wait,
        },
        {
          mintAddress,
          maxAttempts: 6,
          intervalMs: 700,
        },
      );

      if (mountedRef.current) {
        setResolvedTransferredToken(resolved);
      }

      return resolved;
    } finally {
      if (mountedRef.current) {
        setResolvingTransferredToken(false);
      }
    }
  }, [resolvingTransferredToken, transferredMintAddress]);

  const openContentsAfterResolve = useCallback(async () => {
    if (!transferSuccessModalViewModel) {
      return;
    }

    const searchParams = new URLSearchParams({
      mintAddress: transferSuccessModalViewModel.mintAddress,
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
  }, [closeTransferModal, navigate, transferSuccessModalViewModel]);

  const openTokenContentsByMintAddress = useCallback(
    async (mintAddress: string) => {
      const normalizedMintAddress = mintAddress.trim();

      if (!normalizedMintAddress) {
        return;
      }

      const headers = await getAuthHeadersOrUndefined();

      if (!hasAuthorization(headers)) {
        navigate("/signin");
        return;
      }

      const resolved = await resolveOwnedWalletTokenByMintAddress(
        normalizedMintAddress,
        headers,
      );

      const fallbackToken = previewState?.raw.token ?? null;
      const fallbackTokenBlueprintPatch = previewState?.tokenBlueprintPatch ?? null;
      const fallbackProductBlueprintPatch =
        previewState?.raw.productBlueprintPatch ?? null;

      const metadataUri =
        resolved.metadataUri.trim() || fallbackToken?.metadataUri.trim() || "";

      const tokenBlueprintId =
        resolved.tokenBlueprintId.trim() ||
        fallbackToken?.tokenBlueprintId.trim() ||
        fallbackTokenBlueprintPatch?.id.trim() ||
        "";

      if (!metadataUri) {
        return;
      }

      const resolvedMintAddress =
        resolved.mintAddress.trim() || normalizedMintAddress;

      const resolvedProductId = resolved.productId.trim() || productId.trim();

      const resolvedBrandId =
        resolved.brandId.trim() || fallbackToken?.brandId.trim() || "";

      const resolvedBrandName =
        resolved.brandName.trim() || fallbackToken?.brandName.trim() || "";

      const resolvedProductName =
        resolved.productName.trim() ||
        fallbackProductBlueprintPatch?.productName ||
        "";

      const productBlueprintIdForContents =
        resolved.productBlueprintId.trim() ||
        previewState?.raw.productBlueprintId.trim() ||
        "";

      const tokenName = fallbackTokenBlueprintPatch?.tokenName || "";
      const tokenIconUrl = fallbackTokenBlueprintPatch?.tokenIcon || "";

      const searchParams = new URLSearchParams();

      searchParams.set("mintAddress", resolvedMintAddress);
      searchParams.set("metadataUri", metadataUri);

      if (resolvedProductId) {
        searchParams.set("productId", resolvedProductId);
      }

      if (resolvedBrandId) {
        searchParams.set("brandId", resolvedBrandId);
      }

      if (resolvedBrandName) {
        searchParams.set("brandName", resolvedBrandName);
      }

      if (resolvedProductName) {
        searchParams.set("productName", resolvedProductName);
      }

      if (productBlueprintIdForContents) {
        searchParams.set("productBlueprintId", productBlueprintIdForContents);
      }

      if (tokenBlueprintId) {
        searchParams.set("tokenBlueprintId", tokenBlueprintId);
      }

      if (tokenName) {
        searchParams.set("tokenName", tokenName);
      }

      if (tokenIconUrl) {
        searchParams.set("tokenIconUrl", tokenIconUrl);
      }

      navigate(`/contents?${searchParams.toString()}`);
    },
    [navigate, previewState, productId],
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
            getAuthHeadersOrUndefined,
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
      } catch (e) {
        setPostReviewError(toScanReviewErrorMessage(e));

        return false;
      } finally {
        if (mountedRef.current) {
          setPostingReview(false);
        }
      }
    },
    [loadReviews, postingReview, productBlueprintId],
  );

  const manualTransfer = useCallback(async () => {
    const pid = productId.trim();

    if (!pid) {
      return;
    }

    setBusyTransfer(true);
    setError(null);
    setTransferModalError(null);

    try {
      const headers = await getAuthHeadersOrUndefined();

      if (!hasAuthorization(headers)) {
        navigate("/signin");
        return;
      }

      const result = await runScanAutoTransfer(
        {
          fetchMeWallet,
          transferScanPurchased,
        },
        {
          productId: pid,
          headers,
        },
      );

      if (result.transferredMintAddress) {
        setTransferredMintOverride(result.transferredMintAddress);
      }

      setTransferResult(result.transferResult);
      setVerifyResult(verifyResultFromTransferResult(result.transferResult));

      if (result.transferResult.matched) {
        setTransferModalError(null);
        setTransferModalOpen(true);
      } else {
        setTransferModalError(
          "この商品は現在のアバターに紐づく受け取り対象ではありません。",
        );
        setTransferModalOpen(true);
      }
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);

      setError(message);
      setTransferModalError(message);
      setTransferModalOpen(true);
    } finally {
      if (mountedRef.current) {
        setBusyTransfer(false);
      }
    }
  }, [navigate, productId]);

  const nextReviewsPage = useCallback(async () => {
    if (busyReviews) {
      return;
    }

    if (reviews?.hasNext !== true) {
      return;
    }

    await loadReviews(reviewPage + 1);
  }, [busyReviews, loadReviews, reviewPage, reviews?.hasNext]);

  const prevReviewsPage = useCallback(async () => {
    if (busyReviews) {
      return;
    }

    if (reviewPage <= 1) {
      return;
    }

    await loadReviews(reviewPage - 1);
  }, [busyReviews, loadReviews, reviewPage]);

  useEffect(() => {
    mountedRef.current = true;
    void load();

    return () => {
      mountedRef.current = false;
    };

    // Intentionally depend only on productId.
    // Depending on load causes repeated preview/avatar/wallet/transfer requests
    // because load is recreated after transfer-related state updates.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [productId]);

  useEffect(() => {
    if (!productBlueprintId) {
      return;
    }

    void loadReviews(1);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [productBlueprintId]);

  useEffect(() => {
    if (!previewMintAddress) {
      return;
    }

    void loadOwnedState();

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [previewMintAddress]);

  return {
    state,
    viewModel,
    transferSuccessModalViewModel,
    displayTransfers,
    hasMultipleTransfers,
    load,
    loadReviews,
    loadOwnedState,
    submitReview,
    manualTransfer,
    nextReviewsPage,
    prevReviewsPage,
    openContentsAfterResolve,
    openTokenContentsByMintAddress,
    transferModalOpen,
    transferModalError,
    closeTransferModal,
  };
}