// frontend/amol/src/features/scan-result/infrastructure/scanResultApi.ts

import { HttpError, requestJson } from "../../../lib/http";
import { getOptionalAuthHeaders } from "../../../lib/authHeaders";

import type {
  ProductBlueprintReview,
  ProductBlueprintReviewPage,
} from "../../shared/types/review";

import type {
  MallPreviewResponse,
  MallScanTransferResponse,
  PreviewState,
} from "../../shared/types/scanResult";

export type WalletResolvedTokenResponse = {
  productId: string;
  brandId: string;
  brandName: string;
  productBlueprintId: string;
  productName: string;
  metadataUri: string;
  assetId: string;
};

export async function loadPreviewState(productId: string): Promise<PreviewState> {
  const id = productId.trim();

  if (!id) {
    throw new Error("preview: productId is empty");
  }

  const authHeaders = await getOptionalAuthHeaders();
  const path = authHeaders ? "/mall/me/preview" : "/mall/preview";

  const raw = await requestJson<MallPreviewResponse>(path, {
    method: "GET",
    headers: authHeaders,
    query: { productId: id },
    unwrapData: true,
    messages: {
      requestErrorMessage: "fetchPreviewByProductId failed",
      nonJsonErrorMessage: "fetchPreviewByProductId failed: response is not json",
      invalidJsonErrorMessage: "fetchPreviewByProductId failed: invalid json",
    },
  });

  return {
    raw,
    tokenIconUrlEncoded: raw.tokenBlueprintPatch?.tokenIcon ?? null,
  };
}

export async function transferScanPurchased(args: {
  productId: string;
  operationId: string;
  headers?: HeadersInit;
}): Promise<MallScanTransferResponse> {
  const productId = args.productId.trim();
  const operationId = args.operationId.trim();

  if (!productId) {
    throw new Error("productId is empty");
  }

  if (!operationId) {
    throw new Error("operationId is empty");
  }

  const headers = new Headers(args.headers);
  headers.set("Idempotency-Key", operationId);

  return requestJson<MallScanTransferResponse>(
    "/mall/me/orders/scan/transfer",
    {
      method: "POST",
      auth: "required",
      headers,
      json: { productId },
      unwrapData: true,
      messages: {
        requestErrorMessage: "transferScanPurchased failed",
        nonJsonErrorMessage: "transferScanPurchased failed: response is not json",
        invalidJsonErrorMessage: "transferScanPurchased failed: invalid json",
      },
    },
  );
}

export async function fetchReviewsByProductBlueprintId(args: {
  productBlueprintId: string;
  page: number;
  perPage: number;
}): Promise<ProductBlueprintReviewPage> {
  const productBlueprintId = args.productBlueprintId.trim();

  if (!productBlueprintId) {
    throw new Error("preview review: productBlueprintId is empty");
  }

  return requestJson<ProductBlueprintReviewPage>(
    `/mall/catalog/product-blueprints/${encodeURIComponent(productBlueprintId)}/reviews`,
    {
      method: "GET",
      auth: "none",
      query: {
        page: args.page,
        perPage: args.perPage,
      },
      messages: {
        requestErrorMessage: "fetchReviewsByProductBlueprintId failed",
        nonJsonErrorMessage: "fetchReviewsByProductBlueprintId failed: response is not json",
        invalidJsonErrorMessage: "fetchReviewsByProductBlueprintId failed: invalid json",
      },
    },
  );
}

export async function createProductBlueprintReview(args: {
  productBlueprintId: string;
  body: string;
  rating: number;
  title?: string;
  headers?: HeadersInit;
}): Promise<ProductBlueprintReview> {
  const productBlueprintId = args.productBlueprintId.trim();
  const body = args.body.trim();

  if (!productBlueprintId) {
    throw new Error("preview review create: productBlueprintId is empty");
  }

  if (!body) {
    throw new Error("preview review create: body is empty");
  }

  const rating = Math.max(1, Math.min(5, Math.trunc(args.rating)));
  const title = args.title?.trim() || "Review";

  return requestJson<ProductBlueprintReview>(
    `/mall/me/catalog/product-blueprints/${encodeURIComponent(productBlueprintId)}/reviews`,
    {
      method: "POST",
      auth: "required",
      headers: args.headers,
      json: {
        body,
        rating,
        title,
      },
      messages: {
        requestErrorMessage: "createProductBlueprintReview failed",
        nonJsonErrorMessage: "createProductBlueprintReview failed: response is not json",
        invalidJsonErrorMessage: "createProductBlueprintReview failed: invalid json",
      },
    },
  );
}

export async function resolveOwnedWalletTokenByAssetId(
  assetId: string,
  headers?: HeadersInit,
): Promise<WalletResolvedTokenResponse> {
  const id = assetId.trim();

  if (!id) {
    throw new Error("assetId is empty");
  }

  return requestJson<WalletResolvedTokenResponse>(
    "/mall/me/wallets/tokens/resolve",
    {
      method: "GET",
      auth: "required",
      headers,
      query: { assetId: id },
      messages: {
        requestErrorMessage: "resolveOwnedWalletTokenByAssetId failed",
        nonJsonErrorMessage: "resolveOwnedWalletTokenByAssetId failed: response is not json",
        invalidJsonErrorMessage: "resolveOwnedWalletTokenByAssetId failed: invalid json",
      },
    },
  );
}

export async function isOwnedByWalletAssetId(
  assetId: string,
  headers?: HeadersInit,
): Promise<boolean> {
  const id = assetId.trim();

  if (!id) {
    return false;
  }

  try {
    await requestJson<WalletResolvedTokenResponse>(
      "/mall/me/wallets/tokens/resolve",
      {
        method: "GET",
        auth: "required",
        headers,
        query: { assetId: id },
      },
    );

    return true;
  } catch (error) {
    if (
      error instanceof HttpError &&
      (error.status === 403 || error.status === 404)
    ) {
      return false;
    }

    throw error;
  }
}