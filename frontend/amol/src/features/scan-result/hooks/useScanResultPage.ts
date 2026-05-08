// frontend/amol/src/features/scan-result/hooks/useScanResultPage.ts
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import {
  createProductBlueprintReview,
  fetchMeAvatar,
  fetchMeWallet,
  fetchReviewsByProductBlueprintId,
  getAuthHeadersOrUndefined,
  isOwnedByWalletMintAddress,
  listSolanaTransfersByMintAddress,
  loadPreviewState,
  resolveOwnedWalletTokenByMintAddress,
  resolveTokenByMintAddress,
  transferScanPurchased,
} from "../api/scanResultApi";
import type {
  CatalogReviewPage,
  MallOwnerInfo,
  MallPreviewTransferInfo,
  MallScanTransferResponse,
  MallScanVerifyResponse,
  PreviewState,
  ScanResultPageState,
  TokenResolveDTO,
} from "../types";

function extractNonEmptyTokens(tokens: string[] | null | undefined): Set<string> {
  const out = new Set<string>();

  (tokens ?? []).forEach((token) => {
    const s = token.trim();
    if (s) out.add(s);
  });

  return out;
}

function getDifference(after: Set<string>, before: Set<string>): string[] {
  return [...after].filter((value) => !before.has(value));
}

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function hasAuthorization(headers?: HeadersInit): boolean {
  if (!headers) return false;

  const h = new Headers(headers);
  return Boolean((h.get("Authorization") || h.get("authorization") || "").trim());
}

function verifyResultFromTransferResult(
  result: MallScanTransferResponse
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
  const [chainTransfers, setChainTransfers] = useState<MallPreviewTransferInfo[]>(
    []
  );

  const [reviews, setReviews] = useState<CatalogReviewPage | null>(null);
  const [reviewsError, setReviewsError] = useState<string | null>(null);
  const [reviewPage, setReviewPage] = useState(1);
  const [reviewPerPage] = useState(20);
  const [busyReviews, setBusyReviews] = useState(false);

  const [ownedByWallet, setOwnedByWallet] = useState<boolean | null>(null);
  const [ownedByWalletError, setOwnedByWalletError] = useState<string | null>(
    null
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

  const displayTransfers = useMemo(() => {
    const backendTransfers = previewState?.raw.transfers ?? [];
    return backendTransfers.length > 0 ? backendTransfers : chainTransfers;
  }, [chainTransfers, previewState?.raw.transfers]);

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

  const runAutoTransferIfNeeded = useCallback(
    async (pid: string, headers?: HeadersInit) => {
      const normalizedProductId = pid.trim();
      if (!normalizedProductId) return;
      if (!hasAuthorization(headers)) return;
      if (autoTransferTriggeredRef.current || transferringRef.current) return;

      autoTransferTriggeredRef.current = true;
      transferringRef.current = true;
      setBusyTransfer(true);

      try {
        let beforeTokens = new Set<string>();

        try {
          const w0 = await fetchMeWallet(headers);
          beforeTokens = extractNonEmptyTokens(w0.tokens);
        } catch {
          beforeTokens = new Set<string>();
        }

        const result = await transferScanPurchased({
          productId: normalizedProductId,
          headers,
        });

        const transferMint = result.mintAddress.trim();
        if (transferMint) {
          setTransferredMintOverride(transferMint);
        }

        try {
          const w1 = await fetchMeWallet(headers);
          const after = extractNonEmptyTokens(w1.tokens);
          const added = getDifference(after, beforeTokens);

          if (!transferMint && added.length > 0) {
            setTransferredMintOverride(added[0]);
          }
        } catch {
          // noop
        }

        if (!mountedRef.current) return;

        setTransferResult(result);
        setVerifyResult(verifyResultFromTransferResult(result));
      } catch {
        // Auto transfer is best-effort.
        // Backend owns verification / ownership / purchase checks.
      } finally {
        transferringRef.current = false;
        if (mountedRef.current) setBusyTransfer(false);
      }
    },
    []
  );

  const loadAuthFlow = useCallback(
    async (pid: string) => {
      const normalizedProductId = pid.trim();
      if (!normalizedProductId) return;

      const headers = await getAuthHeadersOrUndefined();
      const hasAuth = hasAuthorization(headers);
      setAuthAvailable(hasAuth);

      if (!hasAuth) return;

      try {
        const avatar = await fetchMeAvatar(headers);
        if (!mountedRef.current) return;

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
    [runAutoTransferIfNeeded]
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
    setReviews(null);
    setReviewsError(null);
    setOwnedByWallet(null);
    setOwnedByWalletError(null);
    setPostReviewError(null);
    setReviewPage(1);
    setChainTransfers([]);
    autoTransferTriggeredRef.current = false;
    transferringRef.current = false;

    try {
      if (!pid) throw new Error("商品ID が無いため、プレビューを取得しません。");

      loadingProductIdRef.current = pid;

      const nextState = await loadPreviewState(pid);
      if (!mountedRef.current || loadingProductIdRef.current !== pid) return;

      setPreviewState(nextState);

      const mintAddress = nextState.raw.token?.mintAddress.trim() || "";
      if (mintAddress) {
        try {
          const transfers = await listSolanaTransfersByMintAddress({
            mintAddress,
            limit: 50,
          });

          if (mountedRef.current && loadingProductIdRef.current === pid) {
            setChainTransfers(transfers);
          }
        } catch {
          if (mountedRef.current && loadingProductIdRef.current === pid) {
            setChainTransfers([]);
          }
        }
      }

      await loadAuthFlow(pid);
    } catch (e) {
      if (!mountedRef.current) return;
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (mountedRef.current) setLoading(false);
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
        const res = await fetchReviewsByProductBlueprintId({
          productBlueprintId: pbId,
          page: nextPage,
          perPage: reviewPerPage,
        });

        if (!mountedRef.current) return;

        setReviews(res);
        setReviewsError(null);
        setReviewPage(nextPage);
      } catch (e) {
        if (!mountedRef.current) return;

        setReviews(null);
        setReviewsError(e instanceof Error ? e.message : String(e));
      } finally {
        if (mountedRef.current) setBusyReviews(false);
      }
    },
    [busyReviews, productBlueprintId, reviewPage, reviewPerPage]
  );

  const loadOwnedState = useCallback(async () => {
    const mintAddress = previewMintAddress.trim();

    if (!mintAddress) {
      setOwnedByWallet(null);
      setOwnedByWalletError(null);
      return;
    }

    if (busyOwnedByWallet) return;

    setBusyOwnedByWallet(true);
    setOwnedByWalletError(null);

    try {
      const headers = await getAuthHeadersOrUndefined();

      if (!hasAuthorization(headers)) {
        if (!mountedRef.current) return;

        setOwnedByWallet(null);
        setOwnedByWalletError(null);
        return;
      }

      const owned = await isOwnedByWalletMintAddress(mintAddress, headers);

      if (!mountedRef.current) return;

      setOwnedByWallet(owned);
      setOwnedByWalletError(null);
    } catch (e) {
      if (!mountedRef.current) return;

      setOwnedByWallet(null);
      setOwnedByWalletError(e instanceof Error ? e.message : String(e));
    } finally {
      if (mountedRef.current) setBusyOwnedByWallet(false);
    }
  }, [busyOwnedByWallet, previewMintAddress]);

  const resolveTransferredTokenWithoutSync = useCallback(async () => {
    if (resolvingTransferredToken) {
      throw new Error("resolve is already running");
    }

    const mint = transferredMintAddress.trim();
    if (!mint) {
      throw new Error("transferred mintAddress is empty");
    }

    setResolvingTransferredToken(true);

    try {
      const headers = await getAuthHeadersOrUndefined();
      let resolved: TokenResolveDTO | null = null;
      let lastError: unknown = null;

      for (let i = 0; i < 6; i += 1) {
        try {
          resolved = await resolveTokenByMintAddress(mint, headers);

          const files = resolved.tokenContentsFiles.filter(
            (file) => file.isPreviewable && file.viewUri.trim()
          );

          if (files.length > 0) {
            setResolvedTransferredToken(resolved);
            return resolved;
          }

          lastError = new Error("resolved token has no signed contents");
        } catch (e) {
          lastError = e;
        }

        if (i < 5) {
          await new Promise((resolve) => window.setTimeout(resolve, 700));
        }
      }

      throw lastError instanceof Error
        ? lastError
        : new Error("resolve token failed");
    } finally {
      if (mountedRef.current) setResolvingTransferredToken(false);
    }
  }, [resolvingTransferredToken, transferredMintAddress]);

  const openContentsAfterResolve = useCallback(async () => {
    const mint = transferredMintAddress.trim();
    if (!mint) return;

    const resolved = await resolveTransferredTokenWithoutSync();
    const file = resolved.tokenContentsFiles.find(
      (item) => item.isPreviewable && item.viewUri.trim()
    );

    if (file?.viewUri) {
      window.open(file.viewUri, "_blank", "noopener,noreferrer");
      return;
    }

    const params = new URLSearchParams({
      mintAddress: mint,
      from: "preview_transfer",
    });

    if (productId.trim()) {
      params.set("productId", productId.trim());
    }

    navigate(`/wallet/contents?${params.toString()}`);
  }, [
    navigate,
    productId,
    resolveTransferredTokenWithoutSync,
    transferredMintAddress,
  ]);

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
        headers
      );

      const metadataUri = resolved.metadataUri.trim();
      const tokenBlueprintId = resolved.tokenBlueprintId.trim();

      if (!metadataUri || !tokenBlueprintId) {
        return;
      }

      const searchParams = new URLSearchParams();

      searchParams.set("mintAddress", normalizedMintAddress);

      if (resolved.productId.trim()) {
        searchParams.set("productId", resolved.productId.trim());
      }

      if (resolved.brandId.trim()) {
        searchParams.set("brandId", resolved.brandId.trim());
      }

      if (resolved.brandName.trim()) {
        searchParams.set("brandName", resolved.brandName.trim());
      }

      if (resolved.productName.trim()) {
        searchParams.set("productName", resolved.productName.trim());
      }

      if (resolved.productBlueprintId.trim()) {
        searchParams.set(
          "productBlueprintId",
          resolved.productBlueprintId.trim()
        );
      }

      searchParams.set("tokenBlueprintId", tokenBlueprintId);
      searchParams.set("metadataUri", metadataUri);

      navigate(`/contents?${searchParams.toString()}`);
    },
    [navigate]
  );

  const submitReview = useCallback(
    async (body: string, rating: number) => {
      const pbId = productBlueprintId.trim();
      const reviewBody = body.trim();

      if (!reviewBody) {
        setPostReviewError("本文を入力してください");
        return false;
      }

      if (!pbId) {
        setPostReviewError("productBlueprintId が取得できませんでした");
        return false;
      }

      if (postingReview) return false;

      setPostingReview(true);
      setPostReviewError(null);

      try {
        const headers = await getAuthHeadersOrUndefined();

        await createProductBlueprintReview({
          productBlueprintId: pbId,
          body: reviewBody,
          rating,
          title: "Review",
          headers,
        });

        await loadReviews(1);
        setPostReviewError(null);
        return true;
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);

        if (msg.includes("verified purchase required") || msg.includes("403")) {
          setPostReviewError("購入済み（Verified）の方のみ投稿できます");
        } else {
          setPostReviewError(msg);
        }

        return false;
      } finally {
        if (mountedRef.current) setPostingReview(false);
      }
    },
    [loadReviews, postingReview, productBlueprintId]
  );

  const manualTransfer = useCallback(async () => {
    const pid = productId.trim();
    if (!pid) return;

    setBusyTransfer(true);
    setError(null);

    try {
      const headers = await getAuthHeadersOrUndefined();

      if (!hasAuthorization(headers)) {
        navigate("/signin");
        return;
      }

      const result = await transferScanPurchased({
        productId: pid,
        headers,
      });

      const transferMint = result.mintAddress.trim();
      if (transferMint) {
        setTransferredMintOverride(transferMint);
      }

      setTransferResult(result);
      setVerifyResult(verifyResultFromTransferResult(result));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (mountedRef.current) setBusyTransfer(false);
    }
  }, [navigate, productId]);

  const nextReviewsPage = useCallback(async () => {
    if (busyReviews) return;
    if (reviews?.hasNext !== true) return;
    await loadReviews(reviewPage + 1);
  }, [busyReviews, loadReviews, reviewPage, reviews?.hasNext]);

  const prevReviewsPage = useCallback(async () => {
    if (busyReviews) return;
    if (reviewPage <= 1) return;
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
    if (!productBlueprintId) return;

    void loadReviews(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [productBlueprintId]);

  useEffect(() => {
    if (!previewMintAddress) return;

    void loadOwnedState();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [previewMintAddress]);

  return {
    state,
    displayTransfers,
    load,
    loadReviews,
    loadOwnedState,
    submitReview,
    manualTransfer,
    nextReviewsPage,
    prevReviewsPage,
    openContentsAfterResolve,
    openTokenContentsByMintAddress,
  };
}