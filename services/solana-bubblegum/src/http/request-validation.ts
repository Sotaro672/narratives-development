// services/solana-bubblegum/src/http/request-validation.ts

import {
  publicKey,
  type PublicKey,
} from "@metaplex-foundation/umi";

import { HttpRequestValidationError } from "./errors.js";

export type MintRequestBody = {
  productId?: unknown;
  tokenBlueprintId?: unknown;
  brandId?: unknown;
  toAddress?: unknown;
  name?: unknown;
  symbol?: unknown;
  metadataUri?: unknown;
};

export type MintEstimateRequestBody = {
  tokenBlueprintId?: unknown;
  mintQuantity?: unknown;
  toAddress?: unknown;
  name?: unknown;
  symbol?: unknown;
};

export type OwnedAssetsRequestBody = {
  assetStandard?: unknown;
  walletAddress?: unknown;
};

export type TransferRequestBody = {
  productId?: unknown;
  assetStandard?: unknown;
  assetId?: unknown;
  fromAvatarId?: unknown;
  fromBrandId?: unknown;
  toAvatarId?: unknown;
  brandId?: unknown;
  modelId?: unknown;
  tokenBlueprintId?: unknown;
  fromWalletAddress?: unknown;
  toWalletAddress?: unknown;
};

function requireRequestBodyObject(value: unknown): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new HttpRequestValidationError(
      "body",
      "JSON object is required",
    );
  }

  return value as Record<string, unknown>;
}

export function readMintRequestBody(value: unknown): MintRequestBody {
  return requireRequestBodyObject(value) as MintRequestBody;
}

export function readMintEstimateRequestBody(
  value: unknown,
): MintEstimateRequestBody {
  return requireRequestBodyObject(value) as MintEstimateRequestBody;
}

export function readOwnedAssetsRequestBody(
  value: unknown,
): OwnedAssetsRequestBody {
  return requireRequestBodyObject(value) as OwnedAssetsRequestBody;
}

export function readTransferRequestBody(
  value: unknown,
): TransferRequestBody {
  return requireRequestBodyObject(value) as TransferRequestBody;
}

export function requiredString(
  field: string,
  value: unknown,
): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new HttpRequestValidationError(
      field,
      "value is required",
    );
  }

  return value;
}

export function optionalString(
  field: string,
  value: unknown,
): string {
  if (value === undefined || value === null) {
    return "";
  }

  if (typeof value !== "string") {
    throw new HttpRequestValidationError(
      field,
      "value must be string",
    );
  }

  return value;
}

export function stringValue(
  field: string,
  value: unknown,
): string {
  if (typeof value !== "string") {
    throw new HttpRequestValidationError(
      field,
      "value must be string",
    );
  }

  return value;
}

export function requiredPositiveInteger(
  field: string,
  value: unknown,
): number {
  if (
    typeof value !== "number" ||
    !Number.isSafeInteger(value) ||
    value <= 0
  ) {
    throw new HttpRequestValidationError(
      field,
      "value must be a positive integer",
    );
  }

  return value;
}

export function parseSolanaPublicKey(
  field: string,
  value: string,
): PublicKey {
  try {
    return publicKey(value);
  } catch (error) {
    const detail =
      error instanceof Error
        ? error.message
        : String(error);

    throw new HttpRequestValidationError(
      field,
      `value must be a valid Solana public key detail=${detail}`,
    );
  }
}