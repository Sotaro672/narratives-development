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

export type TokenBlueprintPatchDTO = {
  id: string;
  tokenName: string;
  symbol: string;
  brandId: string;
  brandName: string;
  companyId: string;
  description: string;
  minted: boolean;
  metadataUri: string;
  iconUrl?: string;
};

type TokenBlueprintPatchRaw = {
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

const toText = (value: unknown): string => {
  return typeof value === "string" ? value.trim() : "";
};

const toBool = (value: unknown): boolean => {
  if (typeof value === "boolean") return value;
  if (typeof value === "string") return value.trim().toLowerCase() === "true";
  return false;
};

const mapTokenBlueprintRaw = (
  tb: TokenBlueprintRaw,
): TokenBlueprintForMintDTO => {
  const tokenName = toText(tb.tokenName) || toText(tb.name);

  return {
    id: toText(tb.id),

    // 右側の一覧表示用
    name: tokenName,

    // TokenBlueprintCard 表示用
    tokenName,
    symbol: toText(tb.symbol),
    brandId: toText(tb.brandId),
    brandName: toText(tb.brandName),
    companyId: toText(tb.companyId),
    description: toText(tb.description),
    minted: toBool(tb.minted),
    metadataUri: toText(tb.metadataUri),
    iconUrl: toText(tb.iconUrl) || undefined,
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

export async function fetchTokenBlueprintPatchHTTP(
  tokenBlueprintId: string,
): Promise<TokenBlueprintPatchDTO | null> {
  const trimmed = String(tokenBlueprintId ?? "").trim();
  if (!trimmed) return null;

  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/mint/token_blueprints/${encodeURIComponent(
    trimmed,
  )}/patch`;

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch tokenBlueprintPatch (mint): ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const raw = (await res.json()) as TokenBlueprintPatchRaw;

  const id = toText(raw.id);
  if (!id) return null;

  return {
    id,
    tokenName: toText(raw.tokenName),
    symbol: toText(raw.symbol),
    brandId: toText(raw.brandId),
    brandName: toText(raw.brandName),
    companyId: toText(raw.companyId),
    description: toText(raw.description),
    minted: toBool(raw.minted),
    metadataUri: toText(raw.metadataUri),
    iconUrl: toText(raw.iconUrl) || undefined,
  };
}