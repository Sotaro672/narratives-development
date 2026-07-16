// frontend/console/sales/infrastructure/sales_repository_http.ts
import { API_BASE } from "../../shell/src/shared/http/apiBase";
import { getAuthJsonHeaders } from "../../shell/src/shared/http/authHeaders";

// ============================================================
// Domain types
// ============================================================

export type SalesOwner = {
  avatarId: string;
};

export type SalesProductBlueprint = {
  productBlueprintId: string;
  productName: string;
};

export type SalesRow = {
  tokenBlueprintId: string;
  tokenName: string;
  brandId: string;
  brandName: string;
  mintAddresses: string[];
  modelIds: string[];
  productBlueprints: SalesProductBlueprint[];
  owners: SalesOwner[];
};

export type SalesQueryResult = {
  companyId: string;
  rows: SalesRow[];
};

// ============================================================
// Endpoint
// ============================================================

const SALES_ENDPOINT = "/sales";

// ============================================================
// HTTP helper
// ============================================================

async function apiGetJson<T>(path: string): Promise<T> {
  const headers = await getAuthJsonHeaders();

  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    headers: {
      ...headers,
      Accept: "application/json",
    },
    credentials: "include",
  });

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(text || `GET ${path} failed: ${res.status}`);
  }

  if (!text) {
    throw new Error(`GET ${path} returned an empty response`);
  }

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(`GET ${path} returned invalid JSON`);
  }
}

// ============================================================
// Repository
// ============================================================

/**
 * GET /sales
 *
 * 対象企業はバックエンドが認証情報から判定するため、
 * companyIdのクエリパラメーターは送信しない。
 */
export async function listSales(): Promise<SalesQueryResult> {
  return apiGetJson<SalesQueryResult>(SALES_ENDPOINT);
}