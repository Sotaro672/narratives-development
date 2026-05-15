// frontend/console/mintRequest/src/infrastructure/repository/http/tokenBlueprints.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { TokenBlueprintForMintDTO } from "../../dto/mintRequestLocal.dto";

type TokenBlueprintRaw = {
  id?: unknown;
  name?: unknown;
  symbol?: unknown;
  iconUrl?: unknown;
};

const toText = (value: unknown): string => {
  return typeof value === "string" ? value.trim() : "";
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
    .map((tb: TokenBlueprintRaw): TokenBlueprintForMintDTO => ({
      id: toText(tb.id),
      name: toText(tb.name),
      symbol: toText(tb.symbol),
      iconUrl: toText(tb.iconUrl) || undefined,
    }))
    .filter((tb: TokenBlueprintForMintDTO) => {
      return Boolean(tb.id && tb.name && tb.symbol);
    });
}