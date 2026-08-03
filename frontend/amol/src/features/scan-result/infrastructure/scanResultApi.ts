// frontend/amol/src/features/scan-result/infrastructure/scanResultApi.ts

import {
  readJsonDataResponse,
  readJsonResponse,
} from "../../../components/utils/apiResponse";
import { isRecord } from "../../../components/utils/typeGuards";
import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";

import { getMyAvatar } from "../../avatar/api/avatarApi";
import {
  fetchMeWalletRaw,
  resolveWalletTokenRaw,
} from "../../wallet/api/walletApiClient";

import type {
  CatalogReviewPage,
  MallOwnerInfo,
  MallScanTransferResponse,
  PreviewState,
  TokenResolveDTO,
  WalletDTO,
} from "../../shared/types/scanResult";

import { safeUrl } from "../utils/format";

import {
  getAuthorizationHeader,
  jsonHeaders,
  jsonPostHeaders,
  mergeHeaders,
} from "./scanResultHttp";

import {
  catalogReviewPageFromJson,
  mallPreviewResponseFromJson,
  mallScanTransferResponseFromJson,
  tokenBlueprintPatchVMFromMap,
  tokenResolveDTOFromJson,
  unwrapData,
  walletDTOFromJson,
  walletResolvedTokenResponseFromJson,
  type WalletResolvedTokenResponse,
} from "./scanResultMappers";

export {
  listSolanaTransfersByMintAddress,
} from "./scanResultSolanaApi";

export type {
  WalletResolvedTokenResponse,
} from "./scanResultMappers";

/**
 * 段階移行用の互換関数。
 *
 * Firebase ID tokenの取得自体は共通のgetFirebaseIdTokenへ集約し、
 * ログインしていない場合やtoken取得に失敗した場合だけundefinedへ変換する。
 */
export async function getAuthHeadersOrUndefined(): Promise<
  Record<string, string> | undefined
> {
  try {
    const token = (
      await getFirebaseIdToken()
    ).trim();

    if (!token) {
      return undefined;
    }

    return {
      Authorization:
        `Bearer ${token}`,
    };
  } catch {
    return undefined;
  }
}

async function fetchPreviewRaw(
  productId: string,
  isMe: boolean,
  headers?: HeadersInit,
): Promise<Record<string, unknown>> {
  const id = productId.trim();

  if (!id) {
    throw new Error(
      "preview: productId is empty",
    );
  }

  const base = getApiBaseUrl();

  if (!base) {
    throw new Error(
      "VITE_API_BASE_URL is not configured",
    );
  }

  const path = isMe
    ? "/mall/me/preview"
    : "/mall/preview";

  const url = new URL(
    `${base}${path}`,
  );

  url.searchParams.set(
    "productId",
    id,
  );

  const label = isMe
    ? "fetchMyPreviewByProductId"
    : "fetchPreviewByProductId";

  const response = await fetch(
    url,
    {
      headers: mergeHeaders(
        jsonHeaders(),
        headers,
      ),
    },
  );

  const decoded =
    await readJsonResponse<unknown>(
      response,
      {
        requestErrorMessage:
          `${label} failed`,
        nonJsonErrorMessage:
          `${label} failed: response is not json url=${url.toString()}`,
        invalidJsonErrorMessage:
          `${label} failed: invalid json url=${url.toString()}`,
      },
    );

  if (
    !isRecord(decoded) ||
    Array.isArray(decoded)
  ) {
    throw new Error(
      "invalid json shape (expected object)",
    );
  }

  return decoded;
}

export async function loadPreviewState(
  productId: string,
): Promise<PreviewState> {
  const authHeaders =
    await getAuthHeadersOrUndefined();

  const isMe = Boolean(
    getAuthorizationHeader(
      authHeaders,
    ),
  );

  const raw = await fetchPreviewRaw(
    productId,
    isMe,
    authHeaders,
  );

  const data =
    mallPreviewResponseFromJson(
      raw,
    );

  const unwrapped =
    unwrapData(raw);

  const tokenBlueprintPatchValue =
    unwrapped.tokenBlueprintPatch;

  const tokenBlueprintPatchMap =
    isRecord(
      tokenBlueprintPatchValue,
    ) &&
    !Array.isArray(
      tokenBlueprintPatchValue,
    )
      ? tokenBlueprintPatchValue
      : null;

  const tokenBlueprintPatch =
    tokenBlueprintPatchVMFromMap(
      tokenBlueprintPatchMap,
    );

  return {
    raw: data,
    tokenBlueprintPatch,
    tokenIconUrlEncoded:
      tokenBlueprintPatch?.tokenIcon.trim()
        ? safeUrl(
            tokenBlueprintPatch.tokenIcon,
          )
        : null,
  };
}

export async function fetchMeAvatar(
  headers?: HeadersInit,
): Promise<MallOwnerInfo> {
  const backendUrl =
    getApiBaseUrl();

  if (!backendUrl) {
    throw new Error(
      "VITE_API_BASE_URL is not configured",
    );
  }

  const authorization =
    getAuthorizationHeader(
      headers,
    );

  if (!authorization) {
    throw new Error(
      "Authorization header is required for /mall/me/avatars",
    );
  }

  const idToken = authorization
    .replace(
      /^Bearer\s+/i,
      "",
    )
    .trim();

  if (!idToken) {
    throw new Error(
      "Firebase ID token is required for /mall/me/avatars",
    );
  }

  const avatar =
    await getMyAvatar({
      backendUrl,
      idToken,
    });

  if (!avatar) {
    throw new Error(
      "ログイン中のアバター情報が見つかりません。",
    );
  }

  const avatarWithBrand =
    avatar as typeof avatar & {
      brandId?: unknown;
      brandName?: unknown;
    };

  return {
    brandId:
      typeof avatarWithBrand.brandId ===
      "string"
        ? avatarWithBrand.brandId
        : "",
    avatarId:
      avatar.avatarId,
    brandName:
      typeof avatarWithBrand.brandName ===
      "string"
        ? avatarWithBrand.brandName
        : "",
    avatarName:
      avatar.avatarName,
  };
}

export async function transferScanPurchased(
  args: {
    productId: string;
    headers?: HeadersInit;
  },
): Promise<MallScanTransferResponse> {
  const productId =
    args.productId.trim();

  if (!productId) {
    throw new Error(
      "productId is empty",
    );
  }

  const base =
    getApiBaseUrl();

  if (!base) {
    throw new Error(
      "VITE_API_BASE_URL is not configured",
    );
  }

  const url =
    `${base}/mall/me/orders/scan/transfer`;

  const headers =
    mergeHeaders(
      jsonPostHeaders(),
      args.headers,
    );

  const authHeader =
    getAuthorizationHeader(
      headers,
    );

  if (!authHeader) {
    throw new Error(
      "Authorization header is required for transfer",
    );
  }

  headers.set(
    "Authorization",
    authHeader,
  );

  const response =
    await fetch(
      url,
      {
        method: "POST",
        headers,
        body: JSON.stringify({
          productId,
        }),
      },
    );

  const decoded =
    await readJsonDataResponse<unknown>(
      response,
      {
        requestErrorMessage:
          "transferScanPurchased failed",
        nonJsonErrorMessage:
          "transferScanPurchased failed: response is not json",
        invalidJsonErrorMessage:
          "transferScanPurchased failed: invalid json",
      },
    );

  return mallScanTransferResponseFromJson(
    decoded,
  );
}

export async function fetchReviewsByProductBlueprintId(
  args: {
    productBlueprintId: string;
    page: number;
    perPage: number;
  },
): Promise<CatalogReviewPage> {
  const productBlueprintId =
    args.productBlueprintId.trim();

  if (!productBlueprintId) {
    throw new Error(
      "preview review: productBlueprintId is empty",
    );
  }

  const base =
    getApiBaseUrl();

  if (!base) {
    throw new Error(
      "VITE_API_BASE_URL is not configured",
    );
  }

  const url = new URL(
    `${base}/mall/catalog/product-blueprints/${encodeURIComponent(
      productBlueprintId,
    )}/reviews`,
  );

  url.searchParams.set(
    "page",
    String(args.page),
  );

  url.searchParams.set(
    "perPage",
    String(args.perPage),
  );

  const response =
    await fetch(
      url,
      {
        headers:
          jsonHeaders(),
      },
    );

  const decoded =
    await readJsonDataResponse<unknown>(
      response,
      {
        requestErrorMessage:
          "fetchReviewsByProductBlueprintId failed",
        nonJsonErrorMessage:
          "fetchReviewsByProductBlueprintId failed: response is not json",
        invalidJsonErrorMessage:
          "fetchReviewsByProductBlueprintId failed: invalid json",
      },
    );

  return catalogReviewPageFromJson(
    decoded,
    args.page,
    args.perPage,
  );
}

export async function createProductBlueprintReview(
  args: {
    productBlueprintId: string;
    body: string;
    rating: number;
    title?: string;
    headers?: HeadersInit;
  },
): Promise<Record<string, unknown>> {
  const productBlueprintId =
    args.productBlueprintId.trim();

  const body =
    args.body.trim();

  if (!productBlueprintId) {
    throw new Error(
      "preview review create: productBlueprintId is empty",
    );
  }

  if (!body) {
    throw new Error(
      "preview review create: body is empty",
    );
  }

  const base =
    getApiBaseUrl();

  if (!base) {
    throw new Error(
      "VITE_API_BASE_URL is not configured",
    );
  }

  const rating = Math.max(
    1,
    Math.min(
      5,
      Math.trunc(
        args.rating,
      ),
    ),
  );

  const title =
    args.title?.trim() ||
    "Review";

  const url =
    `${base}/mall/me/catalog/product-blueprints/${encodeURIComponent(
      productBlueprintId,
    )}/reviews`;

  const response =
    await fetch(
      url,
      {
        method: "POST",
        headers:
          mergeHeaders(
            jsonPostHeaders(),
            args.headers,
          ),
        body: JSON.stringify({
          body,
          rating,
          title,
        }),
      },
    );

  const decoded =
    await readJsonDataResponse<unknown>(
      response,
      {
        requestErrorMessage:
          "createProductBlueprintReview failed",
        nonJsonErrorMessage:
          "createProductBlueprintReview failed: response is not json",
        invalidJsonErrorMessage:
          "createProductBlueprintReview failed: invalid json",
        fallbackValue: {},
      },
    );

  if (
    !isRecord(decoded) ||
    Array.isArray(decoded)
  ) {
    throw new Error(
      "invalid json shape (expected object)",
    );
  }

  return decoded;
}

export async function resolveOwnedWalletTokenByMintAddress(
  mintAddress: string,
  headers?: HeadersInit,
): Promise<WalletResolvedTokenResponse> {
  const mint =
    mintAddress.trim();

  if (!mint) {
    throw new Error(
      "mintAddress is empty",
    );
  }

  const result =
    await resolveWalletTokenRaw({
      mintAddress: mint,
      headers,
    });

  if (!result.ok) {
    throw new Error(
      `resolveOwnedWalletTokenByMintAddress failed: ${result.status}`,
    );
  }

  return walletResolvedTokenResponseFromJson(
    result.data,
  );
}

export async function isOwnedByWalletMintAddress(
  mintAddress: string,
  headers?: HeadersInit,
): Promise<boolean> {
  const mint =
    mintAddress.trim();

  if (!mint) {
    return false;
  }

  const result =
    await resolveWalletTokenRaw({
      mintAddress: mint,
      headers,
    });

  if (!result.ok) {
    return false;
  }

  return true;
}

export async function fetchMeWallet(
  headers?: HeadersInit,
): Promise<WalletDTO> {
  const result =
    await fetchMeWalletRaw(
      headers,
    );

  if (!result.ok) {
    throw new Error(
      `fetchMeWallet failed: ${result.status}`,
    );
  }

  return walletDTOFromJson(
    result.data,
  );
}

export async function resolveTokenByMintAddress(
  mintAddress: string,
  headers?: HeadersInit,
): Promise<TokenResolveDTO> {
  const mint =
    mintAddress.trim();

  if (!mint) {
    throw new Error(
      "mintAddress is empty",
    );
  }

  const result =
    await resolveWalletTokenRaw({
      mintAddress: mint,
      headers,
    });

  if (!result.ok) {
    throw new Error(
      `resolveTokenByMintAddress failed: ${result.status}`,
    );
  }

  return tokenResolveDTOFromJson(
    result.data,
    mint,
  );
}