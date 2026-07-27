// frontend/console/shell/src/features/mintRequest/infrastructure/repository/http/tokenBlueprints.ts

import { API_BASE } from "../../../../../shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shared/http/authHeaders";

import type { TokenBlueprintSummary } from "../../../application/port/MintRequestRepository";

type TokenBlueprintRaw = {
  id?: unknown;
  tokenName?: unknown;
  symbol?: unknown;
  brandId?: unknown;
  brandName?: unknown;
  companyId?: unknown;
  description?: unknown;
  minted?: unknown;
  metadataUri?: unknown;
  iconUrl?: unknown;
};

const toText = (
  value: unknown,
): string => {
  return typeof value === "string"
    ? value.trim()
    : "";
};

const toOptionalText = (
  value: unknown,
): string | undefined => {
  const text = toText(value);

  return text || undefined;
};

const toBool = (
  value: unknown,
): boolean => {
  if (typeof value === "boolean") {
    return value;
  }

  if (typeof value === "string") {
    return (
      value.trim().toLowerCase() ===
      "true"
    );
  }

  return false;
};

const mapTokenBlueprintRaw = (
  tokenBlueprint: TokenBlueprintRaw,
): TokenBlueprintSummary => {
  return {
    id: toText(tokenBlueprint.id),
    tokenName: toText(
      tokenBlueprint.tokenName,
    ),
    symbol: toText(
      tokenBlueprint.symbol,
    ),

    brandId: toOptionalText(
      tokenBlueprint.brandId,
    ),
    brandName: toOptionalText(
      tokenBlueprint.brandName,
    ),
    companyId: toOptionalText(
      tokenBlueprint.companyId,
    ),

    description: toOptionalText(
      tokenBlueprint.description,
    ),
    minted: toBool(
      tokenBlueprint.minted,
    ),
    metadataUri: toOptionalText(
      tokenBlueprint.metadataUri,
    ),

    iconUrl: toOptionalText(
      tokenBlueprint.iconUrl,
    ),
  };
};

export async function fetchTokenBlueprintsByBrandHTTP(
  brandId: string,
): Promise<TokenBlueprintSummary[]> {
  const normalizedBrandId =
    String(brandId ?? "").trim();

  if (!normalizedBrandId) {
    return [];
  }

  const authHeaders =
    await getAuthHeadersOrThrow();

  const url =
    `${API_BASE}/mint/token_blueprints` +
    `?brandId=${encodeURIComponent(
      normalizedBrandId,
    )}`;

  const response = await fetch(
    url,
    {
      method: "GET",
      headers: authHeaders,
    },
  );

  if (response.status === 404) {
    return [];
  }

  if (!response.ok) {
    const body = await response
      .text()
      .catch(() => "");

    throw new Error(
      `Failed to fetch tokenBlueprints (mint): ` +
        `${response.status} ${response.statusText}` +
        (
          body
            ? ` body=${body.slice(0, 400)}`
            : ""
        ),
    );
  }

  const responsePayload =
    await response.json() as unknown;

  const rawItems: TokenBlueprintRaw[] =
    Array.isArray(responsePayload)
      ? responsePayload as TokenBlueprintRaw[]
      : [];

  return rawItems
    .map(mapTokenBlueprintRaw)
    .filter(
      (tokenBlueprint) =>
        Boolean(
          tokenBlueprint.id &&
            tokenBlueprint.tokenName &&
            tokenBlueprint.symbol,
        ),
    );
}