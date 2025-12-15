// frontend/console/tokenBlueprint/src/infrastructure/repository/tokenBlueprintRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ドメイン型（UI で使う TokenBlueprint 定義）
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

/**
 * Backend base URL
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as
    | string
    | undefined)?.replace(/\/+$/g, "") ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  return user.getIdToken();
}

// ---------------------------------------------------------
// API レスポンス型
// ---------------------------------------------------------

export interface TokenBlueprintPageResult {
  items: TokenBlueprint[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
}

// ---------------------------------------------------------
// 作成用ペイロード
// ---------------------------------------------------------
export interface CreateTokenBlueprintPayload {
  name: string;
  symbol: string;
  brandId: string;
  companyId?: string;
  description: string;
  assigneeId: string;
  createdBy: string;
  iconId?: string | null;
  contentFiles: string[];
}

// 更新用ペイロード
export interface UpdateTokenBlueprintPayload {
  name?: string;
  symbol?: string;
  brandId?: string;
  description?: string;
  assigneeId?: string;
  iconId?: string | null;
  contentFiles?: string[];
}

// ---------------------------------------------------------
// 内部ヘルパ: レスポンス共通処理
// ---------------------------------------------------------
async function handleJsonResponse<T>(res: Response): Promise<T> {
  const text = await res.text();

  if (!res.ok) {
    try {
      const data = JSON.parse(text);
      const msg = (data && (data.error || data.message)) || res.statusText;
      throw new Error(msg || `HTTP ${res.status}`);
    } catch {
      throw new Error(text || `HTTP ${res.status}`);
    }
  }

  if (!text) {
    return undefined as unknown as T;
  }

  try {
    const parsed = JSON.parse(text);
    return parsed as T;
  } catch {
    return text as unknown as T;
  }
}

function normalizeTokenBlueprint(raw: any): TokenBlueprint {
  // ★ Spread できるように、まず「object として」固定する（TS2698 回避）
  const obj: Record<string, any> =
    raw && typeof raw === "object" ? (raw as Record<string, any>) : {};

  const brandNameRaw = obj.brandName ?? obj.BrandName;
  const brandName =
    brandNameRaw != null ? String(brandNameRaw).trim() : undefined;

  // ★ minted: boolean（未設定/未知値は false）
  const mintedRaw = obj.minted ?? obj.Minted;
  const minted = typeof mintedRaw === "boolean" ? mintedRaw : false;

  return {
    ...(obj as any),
    minted,
    ...(brandName !== undefined ? { brandName } : {}),
  } as TokenBlueprint;
}

function normalizePageResult(raw: any): TokenBlueprintPageResult {
  const obj: Record<string, any> =
    raw && typeof raw === "object" ? (raw as Record<string, any>) : {};

  const rawItems = (obj.items ?? obj.Items ?? []) as any[];
  const items = rawItems.map((it) => normalizeTokenBlueprint(it));

  return {
    items,
    totalCount: (obj.totalCount ?? obj.TotalCount ?? 0) as number,
    totalPages: (obj.totalPages ?? obj.TotalPages ?? 0) as number,
    page: (obj.page ?? obj.Page ?? 1) as number,
    perPage: (obj.perPage ?? obj.PerPage ?? 0) as number,
  };
}

// ---------------------------------------------------------
// Public API
// ---------------------------------------------------------

export async function fetchTokenBlueprints(params?: {
  page?: number;
  perPage?: number;
}): Promise<TokenBlueprintPageResult> {
  const token = await getIdTokenOrThrow();

  const url = new URL(`${API_BASE}/token-blueprints`);
  if (params?.page != null) url.searchParams.set("page", String(params.page));
  if (params?.perPage != null)
    url.searchParams.set("perPage", String(params.perPage));

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });

  const raw = await handleJsonResponse<any>(res);
  return normalizePageResult(raw);
}

/**
 * currentMember.companyId に紐づく一覧
 */
export async function listTokenBlueprintsByCompanyId(
  companyId: string,
): Promise<TokenBlueprint[]> {
  const cid = companyId.trim();
  if (!cid) return [];

  const token = await getIdTokenOrThrow();

  const url = new URL(`${API_BASE}/token-blueprints`);
  url.searchParams.set("perPage", "200");

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });

  const raw = await handleJsonResponse<any>(res);
  const page = normalizePageResult(raw);
  return page.items;
}

/**
 * 詳細取得
 */
export async function fetchTokenBlueprintById(
  id: string,
): Promise<TokenBlueprint> {
  const token = await getIdTokenOrThrow();

  const res = await fetch(
    `${API_BASE}/token-blueprints/${encodeURIComponent(id)}`,
    {
      method: "GET",
      headers: { Authorization: `Bearer ${token}` },
    },
  );

  const raw = await handleJsonResponse<any>(res);
  return normalizeTokenBlueprint(raw);
}

/**
 * 新規作成
 */
export async function createTokenBlueprint(
  payload: CreateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  const token = await getIdTokenOrThrow();

  const body = {
    name: payload.name.trim(),
    symbol: payload.symbol.trim(),
    brandId: payload.brandId.trim(),
    description: payload.description.trim(),
    assigneeId: payload.assigneeId.trim(),
    createdBy: payload.createdBy.trim(),
    iconId: payload.iconId && payload.iconId.trim() ? payload.iconId.trim() : null,
    contentFiles: (payload.contentFiles ?? [])
      .map((x) => x.trim())
      .filter(Boolean),
    companyId: payload.companyId?.trim(),
  };

  const res = await fetch(`${API_BASE}/token-blueprints`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  const raw = await handleJsonResponse<any>(res);
  return normalizeTokenBlueprint(raw);
}

/**
 * 更新
 */
export async function updateTokenBlueprint(
  id: string,
  payload: UpdateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  const token = await getIdTokenOrThrow();

  const body: any = {};

  if (payload.name !== undefined) body.name = payload.name.trim();
  if (payload.symbol !== undefined) body.symbol = payload.symbol.trim();
  if (payload.brandId !== undefined) body.brandId = payload.brandId.trim();
  if (payload.description !== undefined)
    body.description = payload.description.trim();
  if (payload.assigneeId !== undefined)
    body.assigneeId = payload.assigneeId.trim();

  if (payload.iconId !== undefined) {
    if (payload.iconId === null) {
      body.iconId = null;
    } else {
      body.iconId = payload.iconId.trim() ? payload.iconId.trim() : "";
    }
  }

  if (payload.contentFiles !== undefined) {
    body.contentFiles = (payload.contentFiles ?? [])
      .map((x) => x.trim())
      .filter(Boolean);
  }

  const res = await fetch(
    `${API_BASE}/token-blueprints/${encodeURIComponent(id)}`,
    {
      method: "PUT",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    },
  );

  const raw = await handleJsonResponse<any>(res);
  return normalizeTokenBlueprint(raw);
}

/**
 * 削除
 */
export async function deleteTokenBlueprint(id: string): Promise<void> {
  const token = await getIdTokenOrThrow();

  const res = await fetch(
    `${API_BASE}/token-blueprints/${encodeURIComponent(id)}`,
    {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
    },
  );

  await handleJsonResponse<unknown>(res);
}

// ---------------------------------------------------------
// Brand API
// ---------------------------------------------------------

export type BrandSummary = { id: string; name: string };

export async function fetchBrandsForCurrentCompany(): Promise<BrandSummary[]> {
  const token = await getIdTokenOrThrow();

  const url = new URL(`${API_BASE}/brands`);
  url.searchParams.set("perPage", "200");

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });

  const raw = await handleJsonResponse<any>(res);

  const items = (raw?.items ?? raw?.Items ?? []) as any[];

  return items.map((b) => ({
    id: String(b.id ?? b.ID ?? ""),
    name: String(b.name ?? b.Name ?? ""),
  }));
}

export async function fetchBrandNameById(id: string): Promise<string> {
  const trimmed = id.trim();
  if (!trimmed) return "";

  const token = await getIdTokenOrThrow();

  const res = await fetch(`${API_BASE}/brands/${encodeURIComponent(trimmed)}`, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });

  const data = await handleJsonResponse<any>(res);
  return String(data?.name ?? data?.Name ?? "").trim();
}
