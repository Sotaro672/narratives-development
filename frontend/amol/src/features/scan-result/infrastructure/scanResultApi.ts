// frontend/amol/src/features/scan-result/infrastructure/scanResultApi.ts

import {
  requestJson,
} from "../../../lib/http";
import {
  isRecord,
} from "../../../components/utils/typeGuards";
import {
  getApiBaseUrl,
} from "../../../lib/apiBaseUrl";
import {
  getOptionalAuthHeaders,
} from "../../../lib/authHeaders";

import {
  getMyAvatar,
} from "../../avatar/api/avatarApi";
import {
  fetchMeWalletRaw,
  resolveWalletTokenRaw,
} from "../../shared/api";

import type {
  ProductBlueprintReviewPage,
} from "../../shared/types/review";

import type {
  MallOwnerInfo,
  MallScanTransferResponse,
  PreviewState,
} from "../../shared/types/scanResult";

import type {
  ScanWalletSnapshot,
} from "../application/scanTransferUsecase";

import {
  getAuthorizationHeader,
} from "./scanResultHttp";

import {
  catalogReviewPageFromJson,
  mallScanTransferResponseFromJson,
  previewStateFromJson,
  scanWalletSnapshotFromJson,
  walletResolvedTokenResponseFromJson,
  type WalletResolvedTokenResponse,
} from "./scanResultMappers";

export type {
  WalletResolvedTokenResponse,
} from "./scanResultMappers";

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

  const path = isMe
    ? "/mall/me/preview"
    : "/mall/preview";

  const label = isMe
    ? "fetchMyPreviewByProductId"
    : "fetchPreviewByProductId";

  const decoded =
    await requestJson<unknown>(
      path,
      {
        method: "GET",
        headers,
        query: {
          productId: id,
        },
        messages: {
          requestErrorMessage:
            `${label} failed`,
          nonJsonErrorMessage:
            `${label} failed: response is not json`,
          invalidJsonErrorMessage:
            `${label} failed: invalid json`,
        },
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
    await getOptionalAuthHeaders();

  const isMe = Boolean(
    getAuthorizationHeader(
      authHeaders,
    ),
  );

  const raw =
    await fetchPreviewRaw(
      productId,
      isMe,
      authHeaders,
    );

  return previewStateFromJson(
    raw,
  );
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
    .replace(/^Bearer\s+/i, "")
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

  const authHeader =
    getAuthorizationHeader(
      args.headers,
    );

  if (!authHeader) {
    throw new Error(
      "Authorization header is required for transfer",
    );
  }

  const decoded =
    await requestJson<unknown>(
      "/mall/me/orders/scan/transfer",
      {
        method: "POST",
        headers: args.headers,
        json: {
          productId,
        },
        unwrapData: true,
        messages: {
          requestErrorMessage:
            "transferScanPurchased failed",
          nonJsonErrorMessage:
            "transferScanPurchased failed: response is not json",
          invalidJsonErrorMessage:
            "transferScanPurchased failed: invalid json",
        },
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
): Promise<ProductBlueprintReviewPage> {
  const productBlueprintId =
    args.productBlueprintId.trim();

  if (!productBlueprintId) {
    throw new Error(
      "preview review: productBlueprintId is empty",
    );
  }

  const encodedProductBlueprintId =
    encodeURIComponent(
      productBlueprintId,
    );

  const decoded =
    await requestJson<unknown>(
      `/mall/catalog/product-blueprints/${encodedProductBlueprintId}/reviews`,
      {
        method: "GET",
        auth: "none",
        query: {
          page: args.page,
          perPage: args.perPage,
        },
        unwrapData: true,
        messages: {
          requestErrorMessage:
            "fetchReviewsByProductBlueprintId failed",
          nonJsonErrorMessage:
            "fetchReviewsByProductBlueprintId failed: response is not json",
          invalidJsonErrorMessage:
            "fetchReviewsByProductBlueprintId failed: invalid json",
        },
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

  const encodedProductBlueprintId =
    encodeURIComponent(
      productBlueprintId,
    );

  const decoded =
    await requestJson<unknown>(
      `/mall/me/catalog/product-blueprints/${encodedProductBlueprintId}/reviews`,
      {
        method: "POST",
        headers: args.headers,
        json: {
          body,
          rating,
          title,
        },
        unwrapData: true,
        fallbackValue: {},
        messages: {
          requestErrorMessage:
            "createProductBlueprintReview failed",
          nonJsonErrorMessage:
            "createProductBlueprintReview failed: response is not json",
          invalidJsonErrorMessage:
            "createProductBlueprintReview failed: invalid json",
        },
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

  return result.ok;
}

export async function fetchMeWallet(
  headers?: HeadersInit,
): Promise<ScanWalletSnapshot> {
  const result =
    await fetchMeWalletRaw(
      headers,
    );

  if (!result.ok) {
    throw new Error(
      `fetchMeWallet failed: ${result.status}`,
    );
  }

  return scanWalletSnapshotFromJson(
    result.data,
  );
}