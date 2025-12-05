// frontend/console/tokenBlueprint/src/infrastructure/repository/tokenBlueprintRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ドメイン型（UI で使う TokenBlueprint 定義）
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

/**
 * Backend base URL
 * - .env の VITE_BACKEND_BASE_URL を優先
 * - 未設定時は Cloud Run の固定 URL を利用
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

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

/**
 * Go 側 PageResult に対応する型
 * - サーバー側のキーが `items` / `Items` のどちらでも受け取れるように
 *   正規化メソッド側でマッピングします。
 */
export interface TokenBlueprintPageResult {
  items: TokenBlueprint[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
}

// 作成用ペイロード（id / audit 系は backend 側で付与）
export interface CreateTokenBlueprintPayload {
  name: string;
  symbol: string;
  brandId: string;
  companyId?: string; // backend は context の companyId を利用するため任意
  description: string;
  assigneeId: string;
  iconId?: string | null;
  contentFiles: string[];
}

// 更新用ペイロード（部分更新）
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
    // 204 No Content 等
    return undefined as unknown as T;
  }

  return JSON.parse(text) as T;
}

// PageResult 正規化
function normalizePageResult(raw: any): TokenBlueprintPageResult {
  return {
    items: (raw.items ?? raw.Items ?? []) as TokenBlueprint[],
    totalCount: (raw.totalCount ?? raw.TotalCount ?? 0) as number,
    totalPages: (raw.totalPages ?? raw.TotalPages ?? 0) as number,
    page: (raw.page ?? raw.Page ?? 1) as number,
    perPage: (raw.perPage ?? raw.PerPage ?? 0) as number,
  };
}

// ---------------------------------------------------------
// Public API: TokenBlueprint Repository 関数群
// ---------------------------------------------------------

/**
 * 一覧取得（currentMember.companyId でサーバ側が絞り込み）
 * GET /token-blueprints?page=&perPage=
 */
export async function fetchTokenBlueprints(
  params?: { page?: number; perPage?: number },
): Promise<TokenBlueprintPageResult> {
  const token = await getIdTokenOrThrow();

  const url = new URL(`${API_BASE}/token-blueprints`);
  if (params?.page != null) url.searchParams.set("page", String(params.page));
  if (params?.perPage != null)
    url.searchParams.set("perPage", String(params.perPage));

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  const raw = await handleJsonResponse<any>(res);
  return normalizePageResult(raw);
}

/**
 * 詳細取得
 * GET /token-blueprints/:id
 */
export async function fetchTokenBlueprintById(
  id: string,
): Promise<TokenBlueprint> {
  const token = await getIdTokenOrThrow();

  const res = await fetch(
    `${API_BASE}/token-blueprints/${encodeURIComponent(id)}`,
    {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  );

  return handleJsonResponse<TokenBlueprint>(res);
}

/**
 * 新規作成
 * POST /token-blueprints
 */
export async function createTokenBlueprint(
  payload: CreateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  const token = await getIdTokenOrThrow();

  // 空文字は null に正規化（iconId など）
  const body = {
    name: payload.name.trim(),
    symbol: payload.symbol.trim(),
    brandId: payload.brandId.trim(),
    description: payload.description.trim(),
    assigneeId: payload.assigneeId.trim(),
    iconId:
      payload.iconId && payload.iconId.trim()
        ? payload.iconId.trim()
        : null,
    contentFiles: (payload.contentFiles ?? [])
      .map((x) => x.trim())
      .filter(Boolean),
    // companyId は backend が ID トークンから解決するので原則不要だが、
    // 送っても無視されるだけなのであっても良い。
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

  return handleJsonResponse<TokenBlueprint>(res);
}

/**
 * 更新
 * PUT /token-blueprints/:id
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
    body.iconId =
      payload.iconId && payload.iconId.trim()
        ? payload.iconId.trim()
        : "";
    // 空文字は backend 側で「null クリア」として解釈
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

  return handleJsonResponse<TokenBlueprint>(res);
}

/**
 * 削除
 * DELETE /token-blueprints/:id
 */
export async function deleteTokenBlueprint(id: string): Promise<void> {
  const token = await getIdTokenOrThrow();

  const res = await fetch(
    `${API_BASE}/token-blueprints/${encodeURIComponent(id)}`,
    {
      method: "DELETE",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  );

  // 削除系は body が無い前提なので void
  await handleJsonResponse<unknown>(res);
}

// ---------------------------------------------------------
// Brand 取得系（TokenBlueprint 用）
// ---------------------------------------------------------

// Brand 一覧（currentMember.companyId と同じ company のもの）用の簡易型
export type BrandSummary = {
  id: string;
  name: string;
};

/**
 * 現在ログイン中ユーザーと同じ companyId を持つ Brand 一覧を取得
 * GET /brands?perPage=200
 */
export async function fetchBrandsForCurrentCompany(): Promise<BrandSummary[]> {
  const token = await getIdTokenOrThrow();

  const url = new URL(`${API_BASE}/brands`);
  url.searchParams.set("perPage", "200");

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  const raw = await handleJsonResponse<any>(res);
  const items = (raw?.items ?? raw?.Items ?? []) as any[];

  return items.map((b) => ({
    id: String(b.id ?? b.ID ?? ""),
    name: String(b.name ?? b.Name ?? ""),
  }));
}

/**
 * brandId から brandName を取得
 * GET /brands/:id
 */
export async function fetchBrandNameById(id: string): Promise<string> {
  const trimmed = id.trim();
  if (!trimmed) return "";

  const token = await getIdTokenOrThrow();

  const res = await fetch(
    `${API_BASE}/brands/${encodeURIComponent(trimmed)}`,
    {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  );

  const data = await handleJsonResponse<any>(res);
  return String(data?.name ?? data?.Name ?? "").trim();
}
