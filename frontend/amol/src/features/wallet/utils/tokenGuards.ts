// frontend/amol/src/features/wallet/utils/tokenGuards.ts

import type {
  TokenMetadataAttributeDTO,
  TokenMetadataDTO,
  TokenResolveDTO,
  WalletDTO,
} from "../types/tokenTypes";

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function unwrapData(value: unknown): unknown {
  if (!isRecord(value)) {
    return value;
  }

  if (isRecord(value.data)) {
    return value.data;
  }

  return value;
}

export function extractWallet(value: unknown): WalletDTO | null {
  if (!isRecord(value)) {
    return null;
  }

  if (Array.isArray(value.wallets) && value.wallets.length > 0) {
    return toWalletDTO(value.wallets[0]);
  }

  if (isRecord(value.wallet)) {
    return toWalletDTO(value.wallet);
  }

  if ("WalletAddress" in value) {
    return toWalletDTO(value);
  }

  return null;
}

export function toWalletDTO(value: unknown): WalletDTO | null {
  if (!isRecord(value)) {
    return null;
  }

  const walletAddress = getString(value, "WalletAddress");
  const status = getString(value, "Status");
  const lastUpdatedAt = getNullableString(value, "LastUpdatedAt");

  const tokens = Array.isArray(value.Tokens)
    ? value.Tokens.filter((item): item is string => typeof item === "string")
        .map((item) => item.trim())
        .filter(Boolean)
    : [];

  return {
    walletAddress,
    tokens,
    lastUpdatedAt,
    status,
  };
}

export function toTokenMetadataDTO(value: unknown): TokenMetadataDTO | null {
  if (!isRecord(value)) {
    return null;
  }

  const attributes = toTokenMetadataAttributes(value.attributes);
  const tokenBlueprintId = getTokenBlueprintId(attributes);

  return {
    name: getString(value, "name"),
    symbol: getString(value, "symbol"),
    description: getString(value, "description"),
    image: getString(value, "image"),
    externalUrl: getString(value, "external_url"),
    attributes,
    createdAt: getString(value, "created_at"),
    tokenBlueprintId,
    raw: value,
  };
}

export function toTokenResolveDTO(value: unknown): TokenResolveDTO | null {
  if (!isRecord(value)) {
    return null;
  }

  return {
    productId: getString(value, "productId"),
    brandId: getString(value, "brandId"),
    brandName: getString(value, "brandName"),
    productName: getString(value, "productName"),
    metadataUri: getString(value, "metadataUri"),
    mintAddress: getString(value, "mintAddress"),
    productBlueprintId: getString(value, "productBlueprintId"),
    tokenBlueprintId: getString(value, "tokenBlueprintId"),
  };
}

function toTokenMetadataAttributes(
  value: unknown
): TokenMetadataAttributeDTO[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .filter(isRecord)
    .map((item) => ({
      traitType: getString(item, "trait_type"),
      value: getString(item, "value"),
    }))
    .filter((item) => item.traitType || item.value);
}

function getTokenBlueprintId(
  attributes: TokenMetadataAttributeDTO[]
): string {
  const attribute = attributes.find(
    (item) => item.traitType === "TokenBlueprintID"
  );

  return attribute?.value || "";
}

function getString(value: Record<string, unknown>, key: string): string {
  const raw = value[key];

  if (typeof raw !== "string") {
    return "";
  }

  return raw;
}

function getNullableString(
  value: Record<string, unknown>,
  key: string
): string | null {
  const raw = value[key];

  if (raw === null) {
    return null;
  }

  if (typeof raw !== "string") {
    return null;
  }

  return raw;
}