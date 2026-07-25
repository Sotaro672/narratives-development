// frontend/console/mintRequest/src/infrastructure/repository/http/tokenBlueprints.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { TokenBlueprintForMintDTO } from "../../dto/mintRequestLocal.dto";

type TokenBlueprintRaw = {
  id?: unknown;
  name?: unknown;
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

const toText = (value: unknown): string => {
  return typeof value === "string" ? value.trim() : "";
};

const toOptionalText = (value: unknown): string | undefined => {
  const text = toText(value);
  return text || undefined;
};

const toBool = (value: unknown): boolean => {
  if (typeof value === "boolean") return value;

  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    return normalized === "true";
  }

  return false;
};

const mapTokenBlueprintRaw = (
  tb: TokenBlueprintRaw,
): TokenBlueprintForMintDTO => {
  const tokenName = toText(tb.tokenName) || toText(tb.name);

  return {
    id: toText(tb.id),

    // selector 表示用
    name: tokenName,

    // TokenBlueprintCard 表示用
    tokenName,

    symbol: toText(tb.symbol),

    brandId: toOptionalText(tb.brandId),
    brandName: toOptionalText(tb.brandName),
    companyId: toOptionalText(tb.companyId),

    description: toOptionalText(tb.description),
    minted: toBool(tb.minted),
    metadataUri: toOptionalText(tb.metadataUri),

    iconUrl: toOptionalText(tb.iconUrl),
  };
};

export async function fetchTokenBlueprintsByBrandHTTP(
  brandId: string,
): Promise<TokenBlueprintForMintDTO[]> {
  const trimmed = String(brandId ?? "").trim();
  if (!trimmed) return [];

  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/mint/token_blueprints?brandId=${encodeURIComponent(
    trimmed,
  )}`;

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  if (res.status === 404) return [];

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch tokenBlueprints (mint): ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as unknown;

  const rawItems: TokenBlueprintRaw[] = Array.isArray(json)
    ? (json as TokenBlueprintRaw[])
    : [];

  return rawItems
    .map(mapTokenBlueprintRaw)
    .filter((tb: TokenBlueprintForMintDTO) => {
      return Boolean(tb.id && tb.name && tb.symbol);
    });
}