// frontend\console\sales\infrastructure\sales_repository_http.ts
import { API_BASE } from "../../shell/src/shared/http/apiBase";
import { getAuthJsonHeaders } from "../../shell/src/shared/http/authHeaders";

// ============================================================
// Domain types
// ============================================================

export type SalesOwner = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  followerCount: number;
  followingCount: number;
  postCount: number;
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
// API DTOs
// ============================================================

type ApiSalesOwner = {
  avatarId?: string | null;
  avatarName?: string | null;
  avatarIcon?: string | null;
  followerCount?: number | null;
  followingCount?: number | null;
  postCount?: number | null;
};

type ApiSalesProductBlueprint = {
  productBlueprintId?: string | null;
  productName?: string | null;
};

type ApiSalesRow = {
  tokenBlueprintId?: string | null;
  tokenName?: string | null;
  brandId?: string | null;
  brandName?: string | null;
  mintAddresses?: string[] | null;
  modelIds?: string[] | null;
  productBlueprints?: ApiSalesProductBlueprint[] | null;
  owners?: ApiSalesOwner[] | null;
};

type ApiSalesQueryResult = {
  companyId?: string | null;
  rows?: ApiSalesRow[] | null;
};

// ============================================================
// Endpoint
// ============================================================

const SALES_ENDPOINT = "/sales";

// ============================================================
// Helpers
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

  if (!text) return {} as T;

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(text);
  }
}

function uniqueStrings(values: unknown): string[] {
  if (!Array.isArray(values)) return [];

  const seen = new Set<string>();
  const result: string[] = [];

  for (const v of values) {
    const s = String(v ?? "").trim();
    if (!s) continue;
    if (seen.has(s)) continue;

    seen.add(s);
    result.push(s);
  }

  return result;
}

function toSafeNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  const n = Number(value);
  if (!Number.isFinite(n)) {
    return 0;
  }

  return n;
}

function fromApiSalesOwner(owner: ApiSalesOwner): SalesOwner {
  return {
    avatarId: String(owner?.avatarId ?? "").trim(),
    avatarName: String(owner?.avatarName ?? "").trim(),
    avatarIcon: String(owner?.avatarIcon ?? "").trim(),
    followerCount: toSafeNumber(owner?.followerCount),
    followingCount: toSafeNumber(owner?.followingCount),
    postCount: toSafeNumber(owner?.postCount),
  };
}

function fromApiSalesProductBlueprint(
  productBlueprint: ApiSalesProductBlueprint,
): SalesProductBlueprint {
  return {
    productBlueprintId: String(
      productBlueprint?.productBlueprintId ?? "",
    ).trim(),
    productName: String(productBlueprint?.productName ?? "").trim(),
  };
}

function fromApiSalesRow(row: ApiSalesRow): SalesRow {
  const rawOwners = Array.isArray(row?.owners) ? row.owners : [];
  const rawProductBlueprints = Array.isArray(row?.productBlueprints)
    ? row.productBlueprints
    : [];

  return {
    tokenBlueprintId: String(row?.tokenBlueprintId ?? "").trim(),
    tokenName: String(row?.tokenName ?? "").trim(),
    brandId: String(row?.brandId ?? "").trim(),
    brandName: String(row?.brandName ?? "").trim(),
    mintAddresses: uniqueStrings(row?.mintAddresses),
    modelIds: uniqueStrings(row?.modelIds),
    productBlueprints: rawProductBlueprints
      .map(fromApiSalesProductBlueprint)
      .filter((pb) => pb.productBlueprintId !== ""),
    owners: rawOwners
      .map(fromApiSalesOwner)
      .filter((owner) => owner.avatarId !== ""),
  };
}

function fromApiSalesQueryResult(data: ApiSalesQueryResult): SalesQueryResult {
  const rawRows = Array.isArray(data?.rows) ? data.rows : [];

  return {
    companyId: String(data?.companyId ?? "").trim(),
    rows: rawRows
      .map(fromApiSalesRow)
      .filter((row) => row.tokenBlueprintId !== ""),
  };
}

// ============================================================
// Repository
// ============================================================

/**
 * backend: GET /sales
 *
 * companyId は backend 側で middleware.CompanyID(r) から取得する。
 * frontend から query parameter として companyId は送らない。
 */
export async function listSalesByCompanyId(
  _companyId?: string,
): Promise<SalesQueryResult> {
  const data = await apiGetJson<ApiSalesQueryResult>(SALES_ENDPOINT);

  return fromApiSalesQueryResult(data);
}

/**
 * backend: GET /sales
 *
 * companyId を frontend 側で持たない呼び出し用。
 */
export async function listSales(): Promise<SalesQueryResult> {
  const data = await apiGetJson<ApiSalesQueryResult>(SALES_ENDPOINT);

  return fromApiSalesQueryResult(data);
}