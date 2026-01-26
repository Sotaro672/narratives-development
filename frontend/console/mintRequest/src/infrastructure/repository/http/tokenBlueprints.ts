// frontend/console/mintRequest/src/infrastructure/repository/http/tokenBlueprints.ts

import { API_BASE } from "../../http/consoleApiBase";
import { getIdTokenOrThrow } from "../../http/firebaseAuth";
import { buildHeaders } from "../../http/httpClient";
import {
  logHttpRequest,
  logHttpResponse,
  logHttpError,
  safeTokenHint,
} from "../../http/httpLogger";

import type { TokenBlueprintForMintDTO } from "../../dto/mintRequestLocal.dto";
import type {
  TokenBlueprintPageResultDTO,
  TokenBlueprintRecordRaw,
} from "../../dto/mintRequestRaw.dto";

export async function fetchTokenBlueprintsByBrandHTTP(
  brandId: string,
): Promise<TokenBlueprintForMintDTO[]> {
  const trimmed = String(brandId ?? "").trim();
  if (!trimmed) return [];

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/token_blueprints?brandId=${encodeURIComponent(
    trimmed,
  )}`;

  logHttpRequest("fetchTokenBlueprintsByBrandHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: `Bearer ${safeTokenHint(idToken)}`,
      "Content-Type": "application/json",
    },
    brandId: trimmed,
  });

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  logHttpResponse("fetchTokenBlueprintsByBrandHTTP", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (res.status === 404) return [];

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    logHttpError("fetchTokenBlueprintsByBrandHTTP", {
      method: "GET",
      url,
      status: res.status,
      statusText: res.statusText,
      bodyPreview: body ? body.slice(0, 800) : "",
    });
    throw new Error(
      `Failed to fetch tokenBlueprints (mint): ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as
    | TokenBlueprintPageResultDTO
    | TokenBlueprintRecordRaw[]
    | null
    | undefined;

  const rawItems: TokenBlueprintRecordRaw[] = Array.isArray(json)
    ? json
    : json?.items ?? json?.Items ?? [];

  return rawItems
    .map((tb) => ({
      id: String(tb.id ?? tb.ID ?? "").trim(),
      name: String(tb.name ?? tb.Name ?? "").trim(),
      symbol: String(tb.symbol ?? tb.Symbol ?? "").trim(),
      iconUrl: String(tb.iconUrl ?? tb.IconUrl ?? "").trim() || undefined,
    }))
    .filter((tb) => tb.id && tb.name && tb.symbol);
}
