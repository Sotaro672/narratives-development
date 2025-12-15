// frontend/console/tokenBlueprint/src/infrastructure/repository/tokenBlueprintRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ドメイン型（UI で使う TokenBlueprint 定義）
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

/**
 * Backend base URL
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

console.log("[tokenBlueprint/tokenBlueprintRepositoryHTTP] API_BASE resolved =", API_BASE, {
  ENV_BASE,
  usingFallback: !ENV_BASE,
});

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

// optional: actorId (uid) を送る
function getActorIdOrEmpty(): string {
  try {
    return auth.currentUser?.uid?.trim?.() ?? "";
  } catch {
    return "";
  }
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

// ★ 署名付きURL発行レスポンス（Create のレスポンスに embed される想定）
export type SignedIconUpload = {
  uploadUrl: string;
  objectPath: string; // 例: "{tokenBlueprintId}/icon"
  publicUrl: string;  // 例: https://storage.googleapis.com/<bucket>/{tokenBlueprintId}/icon
  expiresAt?: string;
  contentType?: string; // 署名に含まれる。PUT 時に一致必須
};

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
  iconId?: string | null; // 方針Aでは通常未使用（後から objectPath をセットする）
  contentFiles: string[];
}

// 更新用ペイロード
export interface UpdateTokenBlueprintPayload {
  name?: string;
  symbol?: string;
  brandId?: string;
  description?: string;
  assigneeId?: string;
  iconId?: string | null; // 方針A: objectPath を入れる（例: "{docId}/icon"）
  contentFiles?: string[];
}

// ★ Create 時に iconUpload を発行して欲しい場合のオプション（ヘッダで渡す）
export type CreateTokenBlueprintOptions = {
  iconFileName?: string;
  iconContentType?: string; // 空だと発行されない/署名に困るので基本入れる
};

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

  // ★ iconUpload（Create レスポンスで返る）
  const iconUploadRaw = obj.iconUpload ?? obj.IconUpload;
  const iconUpload: SignedIconUpload | undefined =
    iconUploadRaw && typeof iconUploadRaw === "object"
      ? {
          uploadUrl: String(iconUploadRaw.uploadUrl ?? iconUploadRaw.UploadURL ?? "").trim(),
          objectPath: String(iconUploadRaw.objectPath ?? iconUploadRaw.ObjectPath ?? "").trim(),
          publicUrl: String(iconUploadRaw.publicUrl ?? iconUploadRaw.PublicURL ?? "").trim(),
          expiresAt:
            (iconUploadRaw.expiresAt ?? iconUploadRaw.ExpiresAt) != null
              ? String(iconUploadRaw.expiresAt ?? iconUploadRaw.ExpiresAt)
              : undefined,
          contentType:
            (iconUploadRaw.contentType ?? iconUploadRaw.ContentType) != null
              ? String(iconUploadRaw.contentType ?? iconUploadRaw.ContentType).trim()
              : undefined,
        }
      : undefined;

  // iconUrl は handler 側で解決して返してくることがある
  const iconUrlRaw = obj.iconUrl ?? obj.IconURL;
  const iconUrl =
    iconUrlRaw != null ? String(iconUrlRaw).trim() : undefined;

  return {
    ...(obj as any),
    minted,
    ...(brandName !== undefined ? { brandName } : {}),
    ...(iconUpload ? { iconUpload } : {}),
    ...(iconUrl !== undefined ? { iconUrl } : {}),
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
export async function fetchTokenBlueprintById(id: string): Promise<TokenBlueprint> {
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
 *
 * ★ 方針A（推奨）:
 * - create 時点では icon バイナリは送らない
 * - ただし「create レスポンスに iconUpload（署名付きPUT URL）を埋め込んで返す」ために、
 *   options.iconContentType / options.iconFileName をヘッダで渡す
 *
 * フロー例:
 * 1) createTokenBlueprint(payload, { iconFileName, iconContentType })
 * 2) putFileToSignedUrl(tb.iconUpload.uploadUrl, file, tb.iconUpload.contentType)
 * 3) updateTokenBlueprint(tb.id, { iconId: tb.iconUpload.objectPath })
 */
export async function createTokenBlueprint(
  payload: CreateTokenBlueprintPayload,
  options?: CreateTokenBlueprintOptions,
): Promise<TokenBlueprint> {
  const token = await getIdTokenOrThrow();

  const body = {
    name: payload.name.trim(),
    symbol: payload.symbol.trim(),
    brandId: payload.brandId.trim(),
    description: payload.description.trim(),
    assigneeId: payload.assigneeId.trim(),
    createdBy: payload.createdBy.trim(),
    iconId:
      payload.iconId && payload.iconId.trim() ? payload.iconId.trim() : null,
    contentFiles: (payload.contentFiles ?? [])
      .map((x) => x.trim())
      .filter(Boolean),
    companyId: payload.companyId?.trim(),
  };

  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
  };

  // actorId は任意（空でも可）
  const actorId = getActorIdOrEmpty();
  if (actorId) headers["X-Actor-Id"] = actorId;

  // ★ create レスポンスで iconUpload を発行して欲しい場合はヘッダを渡す
  const iconCT = String(options?.iconContentType ?? "").trim();
  const iconFN = String(options?.iconFileName ?? "").trim();

  // handler 側は「どちらか来たら発行」なので、アイコンがある時は両方送るのが安全
  if (iconCT || iconFN) {
    headers["X-Icon-Content-Type"] = iconCT || "application/octet-stream";
    headers["X-Icon-File-Name"] = iconFN || "icon";
  }

  console.log("[tokenBlueprint/tokenBlueprintRepositoryHTTP] createTokenBlueprint request", {
    hasIconHeader: Boolean(iconCT || iconFN),
    iconContentType: headers["X-Icon-Content-Type"],
    iconFileName: headers["X-Icon-File-Name"],
    body: { ...body, contentFiles: (body.contentFiles ?? []).length },
  });

  const res = await fetch(`${API_BASE}/token-blueprints`, {
    method: "POST",
    headers,
    body: JSON.stringify(body),
  });

  const raw = await handleJsonResponse<any>(res);
  const tb = normalizeTokenBlueprint(raw);

  console.log("[tokenBlueprint/tokenBlueprintRepositoryHTTP] createTokenBlueprint response", {
    id: (tb as any)?.id,
    iconUploadIssued: Boolean((tb as any)?.iconUpload?.uploadUrl),
    iconUpload: (tb as any)?.iconUpload,
  });

  return tb;
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

  // iconId:
  // - null: 明示的に削除
  // - "":   backend 側の normalize により NULL 化される想定（既存仕様維持）
  // - 非空: objectPath を入れる（例: "{docId}/icon"）
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

  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
  };

  const actorId = getActorIdOrEmpty();
  if (actorId) headers["X-Actor-Id"] = actorId;

  const res = await fetch(
    `${API_BASE}/token-blueprints/${encodeURIComponent(id)}`,
    {
      method: "PUT",
      headers,
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
// ★ Direct PUT helpers (Front -> Signed URL -> GCS)
// ---------------------------------------------------------

/**
 * ブラウザから署名付きURLへ直接 PUT
 * - 注意: CORS がバケットに設定されていないとブラウザで失敗します
 * - 重要: Content-Type は署名に含まれるので一致必須（backend が返した contentType を優先）
 */
export async function putFileToSignedUrl(
  uploadUrl: string,
  file: File,
  signedContentType?: string,
): Promise<void> {
  const url = String(uploadUrl || "").trim();
  if (!url) throw new Error("uploadUrl is empty");
  if (!file) throw new Error("file is empty");

  const ct = String(signedContentType || "").trim() || file.type || "application/octet-stream";

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": ct,
    },
    body: file,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `GCS PUT failed: ${res.status}`);
  }
}

/**
 * アップロード完了後、TokenBlueprint に icon を紐付ける（方針A）
 * - backend 側に専用 endpoint が無い場合でも、PUT/PATCH で iconId に objectPath を入れればOK
 */
export async function attachTokenBlueprintIcon(params: {
  tokenBlueprintId: string;
  objectPath: string;
}): Promise<TokenBlueprint> {
  const id = params.tokenBlueprintId.trim();
  if (!id) throw new Error("tokenBlueprintId is empty");

  const objectPath = params.objectPath.trim();
  if (!objectPath) throw new Error("objectPath is empty");

  return await updateTokenBlueprint(id, { iconId: objectPath });
}

/**
 * 便利関数:
 * Create レスポンスに含まれる iconUpload を使って「PUT → iconId反映」まで一括実行
 *
 * 例:
 * const tb = await createTokenBlueprint(..., { iconFileName: file.name, iconContentType: file.type })
 * await uploadAndAttachTokenBlueprintIconFromCreateResponse({ tokenBlueprint: tb, file })
 */
export async function uploadAndAttachTokenBlueprintIconFromCreateResponse(params: {
  tokenBlueprint: TokenBlueprint;
  file: File;
}): Promise<TokenBlueprint> {
  const tb: any = params.tokenBlueprint as any;
  const file = params.file;
  if (!file) throw new Error("file is empty");

  const id = String(tb?.id ?? "").trim();
  if (!id) throw new Error("tokenBlueprint.id is empty");

  const upl: SignedIconUpload | undefined = tb?.iconUpload;
  if (!upl?.uploadUrl || !upl?.objectPath) {
    throw new Error(
      "iconUpload is missing on create response. " +
        "Backend env TOKEN_ICON_SIGNER_EMAIL が未設定、または create 時に X-Icon-* ヘッダを付けていない可能性があります。",
    );
  }

  await putFileToSignedUrl(upl.uploadUrl, file, upl.contentType);
  return await attachTokenBlueprintIcon({
    tokenBlueprintId: id,
    objectPath: upl.objectPath,
  });
}

// ---------------------------------------------------------
// （旧）Signed URL 発行 endpoint 方式は未対応
// - 現在の backend 実装は create レスポンスで iconUpload を返す方式です。
// - もし backend に /token-blueprints/:id/icon-upload-url を実装したら復活させてください。
// ---------------------------------------------------------

export type IssueIconUploadUrlPayload = {
  fileName: string;
  contentType: string;
  size?: number;
};

// 互換のため型だけ残す（将来 endpoint ができたら使える）
export type SignedUploadUrlResult = {
  uploadUrl: string;
  objectPath: string;
  publicUrl: string;
  expiresAt?: string;
};

// 現状は backend に endpoint が無い想定なので明示的にエラーにする（404 より分かりやすい）
export async function issueTokenBlueprintIconUploadUrl(_params: {
  tokenBlueprintId: string;
  fileName: string;
  contentType: string;
  size?: number;
}): Promise<SignedUploadUrlResult> {
  throw new Error(
    "issueTokenBlueprintIconUploadUrl is not supported in current backend. " +
      "Use createTokenBlueprint(..., { iconFileName, iconContentType }) and read iconUpload from the create response.",
  );
}

// 現状の uploadAndAttachTokenBlueprintIcon も create レスポンス依存のため非推奨
export async function uploadAndAttachTokenBlueprintIcon(_params: {
  tokenBlueprintId: string;
  file: File;
}): Promise<TokenBlueprint> {
  throw new Error(
    "uploadAndAttachTokenBlueprintIcon is not supported in current backend. " +
      "Use uploadAndAttachTokenBlueprintIconFromCreateResponse({ tokenBlueprint, file }).",
  );
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
