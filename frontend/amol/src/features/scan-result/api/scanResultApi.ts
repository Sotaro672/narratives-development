// frontend/amol/src/features/scan-result/api/scanResultApi.ts
import type {
  CatalogReviewPage,
  MallOwnerInfo,
  MallScanTransferResponse,
  MallScanVerifyResponse,
  PreviewState,
  TokenResolveDTO,
  WalletDTO,
} from "../types";
import { safeUrl } from "../utils/format";
import {
  getAuthHeadersOrUndefined,
  getAuthorizationHeader,
  jsonHeaders,
  jsonPostHeaders,
  mergeHeaders,
  readJsonObject,
  resolveApiBase,
} from "./scanResultHttp";
import {
  catalogReviewPageFromJson,
  mallOwnerInfoFromJson,
  mallPreviewResponseFromJson,
  mallScanTransferResponseFromJson,
  mallScanVerifyResponseFromJson,
  objectOrNull,
  tokenBlueprintPatchVMFromMap,
  tokenResolveDTOFromJson,
  unwrapData,
  walletDTOFromJson,
  walletResolvedTokenResponseFromJson,
  type WalletResolvedTokenResponse,
} from "./scanResultMappers";
export { getAuthHeadersOrUndefined } from "./scanResultHttp";
export { listSolanaTransfersByMintAddress } from "./scanResultSolanaApi";
export type { WalletResolvedTokenResponse } from "./scanResultMappers";

async function fetchPreviewRaw(
  productId: string,
  isMe: boolean,
  headers?: HeadersInit
): Promise<Record<string, unknown>> {
  const id = productId.trim();
  if (!id) throw new Error("preview: productId is empty");

  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const path = isMe ? "/mall/me/preview" : "/mall/preview";
  const url = new URL(`${base}${path}`);

  url.searchParams.set("productId", id);

  return readJsonObject(
    await fetch(url, {
      headers: mergeHeaders(jsonHeaders(), headers),
    }),
    isMe ? "fetchMyPreviewByProductId" : "fetchPreviewByProductId",
    url.toString()
  );
}

export async function loadPreviewState(productId: string): Promise<PreviewState> {
  const authHeaders = await getAuthHeadersOrUndefined();
  const isMe = Boolean(getAuthorizationHeader(authHeaders));

  const raw = await fetchPreviewRaw(productId, isMe, authHeaders);
  const data = mallPreviewResponseFromJson(raw);

  const unwrapped = unwrapData(raw);
  const tbMap = objectOrNull(unwrapped.tokenBlueprintPatch);
  const tokenBlueprintPatch = tokenBlueprintPatchVMFromMap(tbMap);

  return {
    raw: data,
    tokenBlueprintPatch,
    tokenIconUrlEncoded: tokenBlueprintPatch?.tokenIcon.trim()
      ? safeUrl(tokenBlueprintPatch.tokenIcon)
      : null,
  };
}

export async function fetchMeAvatar(headers?: HeadersInit): Promise<MallOwnerInfo> {
  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const url = `${base}/mall/me/avatars`;
  const mergedHeaders = mergeHeaders(jsonHeaders(), headers);

  if (!getAuthorizationHeader(mergedHeaders)) {
    throw new Error("Authorization header is required for /mall/me/avatars");
  }

  const decoded = await readJsonObject(
    await fetch(url, { headers: mergedHeaders }),
    "fetchMeAvatar",
    url
  );

  return mallOwnerInfoFromJson(decoded);
}

export async function verifyScanPurchasedByAvatarId(args: {
  avatarId: string;
  productId: string;
  headers?: HeadersInit;
}): Promise<MallScanVerifyResponse> {
  const avatarId = args.avatarId.trim();
  const productId = args.productId.trim();

  if (!avatarId) throw new Error("avatarId is empty");
  if (!productId) throw new Error("productId is empty");

  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const url = `${base}/mall/me/orders/scan/verify`;

  const decoded = await readJsonObject(
    await fetch(url, {
      method: "POST",
      headers: mergeHeaders(jsonPostHeaders(), args.headers),
      body: JSON.stringify({ avatarId, productId }),
    }),
    "verifyScanPurchasedByAvatarId",
    url
  );

  return mallScanVerifyResponseFromJson(decoded);
}

export async function transferScanPurchased(args: {
  productId: string;
  headers?: HeadersInit;
}): Promise<MallScanTransferResponse> {
  const productId = args.productId.trim();
  if (!productId) throw new Error("productId is empty");

  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const url = `${base}/mall/me/orders/scan/transfer`;
  const headers = mergeHeaders(jsonPostHeaders(), args.headers);
  const authHeader = getAuthorizationHeader(headers);

  if (!authHeader) {
    throw new Error("Authorization header is required for transfer");
  }

  headers.set("Authorization", authHeader);

  const decoded = await readJsonObject(
    await fetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify({ productId }),
    }),
    "transferScanPurchased",
    url
  );

  return mallScanTransferResponseFromJson(decoded);
}

export async function fetchReviewsByProductBlueprintId(args: {
  productBlueprintId: string;
  page: number;
  perPage: number;
}): Promise<CatalogReviewPage> {
  const productBlueprintId = args.productBlueprintId.trim();

  if (!productBlueprintId) {
    throw new Error("preview review: productBlueprintId is empty");
  }

  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const url = new URL(
    `${base}/mall/catalog/product-blueprints/${encodeURIComponent(
      productBlueprintId
    )}/reviews`
  );

  url.searchParams.set("page", String(args.page));
  url.searchParams.set("perPage", String(args.perPage));

  const decoded = await readJsonObject(
    await fetch(url, { headers: jsonHeaders() }),
    "fetchReviewsByProductBlueprintId",
    url.toString()
  );

  return catalogReviewPageFromJson(decoded, args.page, args.perPage);
}

export async function createProductBlueprintReview(args: {
  productBlueprintId: string;
  body: string;
  rating: number;
  title?: string;
  headers?: HeadersInit;
}): Promise<Record<string, unknown>> {
  const productBlueprintId = args.productBlueprintId.trim();
  const body = args.body.trim();

  if (!productBlueprintId) {
    throw new Error("preview review create: productBlueprintId is empty");
  }

  if (!body) {
    throw new Error("preview review create: body is empty");
  }

  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const rating = Math.max(1, Math.min(5, Math.trunc(args.rating)));
  const title = args.title?.trim() || "Review";

  const url = `${base}/mall/me/catalog/product-blueprints/${encodeURIComponent(
    productBlueprintId
  )}/reviews`;

  return readJsonObject(
    await fetch(url, {
      method: "POST",
      headers: mergeHeaders(jsonPostHeaders(), args.headers),
      body: JSON.stringify({ body, rating, title }),
    }),
    "createProductBlueprintReview",
    url
  );
}

export async function resolveOwnedWalletTokenByMintAddress(
  mintAddress: string,
  headers?: HeadersInit
): Promise<WalletResolvedTokenResponse> {
  const mint = mintAddress.trim();
  if (!mint) throw new Error("mintAddress is empty");

  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const url = new URL(`${base}/mall/me/wallets/tokens/resolve`);
  url.searchParams.set("mintAddress", mint);

  const decoded = await readJsonObject(
    await fetch(url, {
      method: "GET",
      headers: mergeHeaders(jsonHeaders(), headers),
    }),
    "resolveOwnedWalletTokenByMintAddress",
    url.toString()
  );

  return walletResolvedTokenResponseFromJson(decoded);
}

export async function isOwnedByWalletMintAddress(
  mintAddress: string,
  headers?: HeadersInit
): Promise<boolean> {
  const mint = mintAddress.trim();
  if (!mint) return false;

  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const url = new URL(`${base}/mall/me/wallets/tokens/resolve`);
  url.searchParams.set("mintAddress", mint);

  const response = await fetch(url, {
    method: "GET",
    headers: mergeHeaders(jsonHeaders(), headers),
  });

  if (response.ok) {
    return true;
  }

  if (response.status === 403 || response.status === 404) {
    return false;
  }

  const text = await response.text();
  const body = text.length > 300 ? text.slice(0, 300) : text;

  throw new Error(
    `isOwnedByWalletMintAddress failed: ${response.status} url=${url.toString()} body=${body}`
  );
}

export async function fetchMeWallet(headers?: HeadersInit): Promise<WalletDTO> {
  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const url = `${base}/mall/me/wallets`;

  const decoded = await readJsonObject(
    await fetch(url, {
      headers: mergeHeaders(jsonHeaders(), headers),
    }),
    "fetchMeWallet",
    url
  );

  return walletDTOFromJson(decoded);
}

export async function resolveTokenByMintAddress(
  mintAddress: string,
  headers?: HeadersInit
): Promise<TokenResolveDTO> {
  const mint = mintAddress.trim();
  if (!mint) throw new Error("mintAddress is empty");

  const base = resolveApiBase();
  if (!base) throw new Error("VITE_API_BASE_URL is not configured");

  const url = new URL(`${base}/mall/me/wallets/tokens/resolve`);
  url.searchParams.set("mintAddress", mint);

  const decoded = await readJsonObject(
    await fetch(url, {
      headers: mergeHeaders(jsonHeaders(), headers),
    }),
    "resolveTokenByMintAddress",
    url.toString()
  );

  return tokenResolveDTOFromJson(decoded, mint);
}