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
 */
export interface TokenBlueprintPageResult {
  items: TokenBlueprint[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
}

// ---------------------------------------------------------
// 作成用ペイロード（createdBy を追加）
// ---------------------------------------------------------
export interface CreateTokenBlueprintPayload {
  name: string;
  symbol: string;
  brandId: string;
  companyId?: string;
  description: string;
  assigneeId: string;
  createdBy: string; // ← 作成者 memberId
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

  // ★ バックエンドから返ってきた生 JSON を確認できるようにログ出力
  try {
    const parsed = JSON.parse(text);
    // ここでは汎用ハンドラなので、どの API かは個々の関数側で分かる前提でラベルのみ
    console.log(
      "[TokenBlueprintRepositoryHTTP] handleJsonResponse raw payload:",
      parsed,
    );
    return parsed as T;
  } catch {
    // JSON でない場合はそのまま text を返す
    console.log(
      "[TokenBlueprintRepositoryHTTP] handleJsonResponse non-JSON text payload:",
      text,
    );
    return text as unknown as T;
  }
}

/**
 * backend から受け取った 1 レコードを TokenBlueprint に正規化
 * - brandName / BrandName を TokenBlueprint.brandName に載せ替え
 */
function normalizeTokenBlueprint(raw: any): TokenBlueprint {
  const tb = raw as TokenBlueprint & {
    BrandName?: string;
  };

  const brandName =
    (raw && (raw.brandName ?? raw.BrandName)) != null
      ? String(raw.brandName ?? raw.BrandName)
      : undefined;

  return {
    ...tb,
    // brandName フィールドを持っていない型でも optional なので代入して問題なし
    ...(brandName !== undefined ? { brandName } : {}),
  };
}

// PageResult 正規化
function normalizePageResult(raw: any): TokenBlueprintPageResult {
  const rawItems = (raw.items ?? raw.Items ?? []) as any[];

  const items = rawItems.map((it) => normalizeTokenBlueprint(it));

  const pageResult: TokenBlueprintPageResult = {
    items,
    totalCount: (raw.totalCount ?? raw.TotalCount ?? 0) as number,
    totalPages: (raw.totalPages ?? raw.TotalPages ?? 0) as number,
    page: (raw.page ?? raw.Page ?? 1) as number,
    perPage: (raw.perPage ?? raw.PerPage ?? 0) as number,
  };

  // ★ 正規化後のページングデータをログ
  console.log("[TokenBlueprintRepositoryHTTP] normalizePageResult:", pageResult);

  return pageResult;
}

// ---------------------------------------------------------
// Public API
// ---------------------------------------------------------

/**
 * 一覧取得（汎用）
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
  console.log(
    "[TokenBlueprintRepositoryHTTP] fetchTokenBlueprints backend raw:",
    raw,
  );

  const page = normalizePageResult(raw);
  console.log(
    "[TokenBlueprintRepositoryHTTP] fetchTokenBlueprints normalized page:",
    page,
  );

  return page;
}

/**
 * currentMember.companyId に紐づくトークン設計一覧を取得
 *
 * - backend 側では AuthMiddleware により context に積まれた companyId を使用して
 *   ListByCompanyID usecase が呼ばれる
 * - ここでは companyId 引数は「空のときは呼ばない」ためのガード用途のみ
 * - ページングは一覧用途として 200 件固定とし、items だけ返す
 */
export async function listTokenBlueprintsByCompanyId(
  companyId: string,
): Promise<TokenBlueprint[]> {
  const cid = companyId.trim();
  if (!cid) return [];

  const token = await getIdTokenOrThrow();

  const url = new URL(`${API_BASE}/token-blueprints`);
  // companyId は backend が context から解決するためクエリでは渡さない
  url.searchParams.set("perPage", "200");

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  const raw = await handleJsonResponse<any>(res);
  console.log(
    "[TokenBlueprintRepositoryHTTP] listTokenBlueprintsByCompanyId backend raw:",
    raw,
  );

  const page = normalizePageResult(raw);
  console.log(
    "[TokenBlueprintRepositoryHTTP] listTokenBlueprintsByCompanyId items:",
    page.items,
  );

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
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  );

  const raw = await handleJsonResponse<any>(res);
  const data = normalizeTokenBlueprint(raw);

  console.log(
    "[TokenBlueprintRepositoryHTTP] fetchTokenBlueprintById backend data:",
    data,
  );

  return data;
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
    createdBy: payload.createdBy.trim(), // backend で createdByName 解決に利用
    iconId:
      payload.iconId && payload.iconId.trim()
        ? payload.iconId.trim()
        : null,
    contentFiles: (payload.contentFiles ?? [])
      .map((x) => x.trim())
      .filter(Boolean),
    // companyId は backend が context から解決するため必須ではないが、
    // 将来の拡張に備えて送ることは許容（backend 側では無視される）
    companyId: payload.companyId?.trim(),
  };

  console.log(
    "[TokenBlueprintRepositoryHTTP] createTokenBlueprint request body:",
    body,
  );

  const res = await fetch(`${API_BASE}/token-blueprints`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  const raw = await handleJsonResponse<any>(res);
  const data = normalizeTokenBlueprint(raw);

  console.log(
    "[TokenBlueprintRepositoryHTTP] createTokenBlueprint backend data:",
    data,
  );

  return data;
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
    body.iconId =
      payload.iconId && payload.iconId.trim()
        ? payload.iconId.trim()
        : "";
  }

  if (payload.contentFiles !== undefined) {
    body.contentFiles = (payload.contentFiles ?? [])
      .map((x) => x.trim())
      .filter(Boolean);
  }

  console.log(
    "[TokenBlueprintRepositoryHTTP] updateTokenBlueprint request body:",
    body,
  );

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
  const data = normalizeTokenBlueprint(raw);

  console.log(
    "[TokenBlueprintRepositoryHTTP] updateTokenBlueprint backend data:",
    data,
  );

  return data;
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
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  );

  await handleJsonResponse<unknown>(res);
  console.log(
    "[TokenBlueprintRepositoryHTTP] deleteTokenBlueprint completed for id:",
    id,
  );
}

// ---------------------------------------------------------
// Brand API
// ---------------------------------------------------------

export type BrandSummary = { id: string; name: string };

/**
 * 一覧取得 — 現在の company の brands
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
  console.log(
    "[TokenBlueprintRepositoryHTTP] fetchBrandsForCurrentCompany backend raw:",
    raw,
  );

  const items = (raw?.items ?? raw?.Items ?? []) as any[];

  const summaries = items.map((b) => ({
    id: String(b.id ?? b.ID ?? ""),
    name: String(b.name ?? b.Name ?? ""),
  }));

  console.log(
    "[TokenBlueprintRepositoryHTTP] fetchBrandsForCurrentCompany summaries:",
    summaries,
  );

  return summaries;
}

/**
 * brandId → brandName
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
  console.log(
    "[TokenBlueprintRepositoryHTTP] fetchBrandNameById backend data:",
    data,
  );

  const name = String(data?.name ?? data?.Name ?? "").trim();
  console.log(
    "[TokenBlueprintRepositoryHTTP] fetchBrandNameById resolved name:",
    { id: trimmed, name },
  );

  return name;
}
