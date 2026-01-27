// frontend/console/mintRequest/src/infrastructure/repository/http/productions.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";
import {
  logHttpError,
  logHttpRequest,
  logHttpResponse,
  safeTokenHint,
} from "../../http/httpLogger";
import {
  normalizeProductionsPayload,
  normalizeProductionIdFromProductionListItem,
  normalizeProductBlueprintIdFromProductionListItem,
} from "../../normalizers/production";

function getAuthValueOrThrow(authHeaders: Record<string, string>): string {
  const authValue = String((authHeaders as any)?.Authorization ?? "").trim();
  if (!authValue) {
    throw new Error("Authorization header is missing (not logged in or token unavailable)");
  }
  return authValue;
}

function extractIdTokenForLog(authValue: string): string {
  const m = String(authValue ?? "").match(/^Bearer\s+(.+)$/i);
  return String(m?.[1] ?? "").trim();
}

/**
 * productionId から productBlueprintId を解決する
 * - primary: GET /productions/{productionId}
 * - fallback: GET /productions を取得してローカル検索
 */
export async function fetchProductBlueprintIdByProductionIdHTTP(
  productionId: string,
): Promise<string | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) throw new Error("productionId が空です");

  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = getAuthValueOrThrow(authHeaders);
  const idToken = extractIdTokenForLog(authValue);

  const url1 = `${API_BASE}/productions/${encodeURIComponent(pid)}`;

  try {
    logHttpRequest("fetchProductBlueprintIdByProductionIdHTTP(primary)", {
      method: "GET",
      url: url1,
      headers: {
        Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
        "Content-Type": "application/json",
      },
    });

    const res1 = await fetch(url1, { method: "GET", headers: authHeaders });

    logHttpResponse("fetchProductBlueprintIdByProductionIdHTTP(primary)", {
      method: "GET",
      url: url1,
      status: res1.status,
      statusText: res1.statusText,
    });

    if (res1.ok) {
      const j1 = (await res1.json()) as any;
      const pb1 = normalizeProductBlueprintIdFromProductionListItem(j1);
      return pb1 ? pb1 : null;
    }
  } catch (e: any) {
    logHttpError("fetchProductBlueprintIdByProductionIdHTTP(primary)", {
      method: "GET",
      url: url1,
      error: String(e?.message ?? e),
    });
    // noop: fallback へ
  }

  const url2 = `${API_BASE}/productions`;

  logHttpRequest("fetchProductBlueprintIdByProductionIdHTTP(fallback)", {
    method: "GET",
    url: url2,
    headers: {
      Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
      "Content-Type": "application/json",
    },
  });

  const res2 = await fetch(url2, { method: "GET", headers: authHeaders });

  logHttpResponse("fetchProductBlueprintIdByProductionIdHTTP(fallback)", {
    method: "GET",
    url: url2,
    status: res2.status,
    statusText: res2.statusText,
  });

  if (!res2.ok) {
    const body = await res2.text().catch(() => "");
    throw new Error(
      `Failed to fetch productions: ${res2.status} ${res2.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json2 = await res2.json();
  const items = normalizeProductionsPayload(json2);

  const hit =
    (items ?? []).find(
      (it: any) => normalizeProductionIdFromProductionListItem(it) === pid,
    ) ?? null;

  const pb2 = hit ? normalizeProductBlueprintIdFromProductionListItem(hit) : "";
  return pb2 ? pb2 : null;
}

/**
 * 現在の company の productions を取得し、productionId の配列を返す（重複除去）
 */
export async function fetchProductionIdsForCurrentCompanyHTTP(): Promise<string[]> {
  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = getAuthValueOrThrow(authHeaders);
  const idToken = extractIdTokenForLog(authValue);

  const url = `${API_BASE}/productions`;

  logHttpRequest("fetchProductionIdsForCurrentCompanyHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
      "Content-Type": "application/json",
    },
  });

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  logHttpResponse("fetchProductionIdsForCurrentCompanyHTTP", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch productions: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = await res.json();
  const items = normalizeProductionsPayload(json);

  const ids: string[] = [];
  const seen = new Set<string>();

  for (const it of items) {
    const pid = normalizeProductionIdFromProductionListItem(it);
    if (!pid || seen.has(pid)) continue;
    seen.add(pid);
    ids.push(pid);
  }

  return ids;
}
