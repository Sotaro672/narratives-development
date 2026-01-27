// frontend/console/mintRequest/src/infrastructure/repository/http/tokenBlueprints.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";
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

  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = String((authHeaders as any)?.Authorization ?? "").trim();
  if (!authValue) {
    throw new Error("Authorization header is missing (not logged in or token unavailable)");
  }

  // For logging only
  const m = authValue.match(/^Bearer\s+(.+)$/i);
  const idToken = (m?.[1] ?? "").trim();

  const url = `${API_BASE}/mint/token_blueprints?brandId=${encodeURIComponent(trimmed)}`;

  logHttpRequest("fetchTokenBlueprintsByBrandHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
      "Content-Type": "application/json",
    },
    brandId: trimmed,
  });

  const res = await fetch(url, { method: "GET", headers: authHeaders });

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
    : (json as any)?.items ?? (json as any)?.Items ?? [];

  return rawItems
    .map((tb) => ({
      id: String((tb as any).id ?? (tb as any).ID ?? "").trim(),
      name: String((tb as any).name ?? (tb as any).Name ?? "").trim(),
      symbol: String((tb as any).symbol ?? (tb as any).Symbol ?? "").trim(),
      iconUrl: String((tb as any).iconUrl ?? (tb as any).IconUrl ?? "").trim() || undefined,
    }))
    .filter((tb) => tb.id && tb.name && tb.symbol);
}
