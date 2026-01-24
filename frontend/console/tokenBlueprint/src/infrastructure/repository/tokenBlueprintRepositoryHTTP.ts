// frontend/console/tokenBlueprint/src/infrastructure/repository/tokenBlueprintRepositoryHTTP.ts

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";
import type { ContentFileDTO } from "../dto/tokenBlueprint.dto";

import { handleJsonResponse } from "../http/json";
import { apiDelete, apiGet, apiPostJson, apiPutJson } from "../http/client";
import { normalizePageResult, normalizeTokenBlueprint } from "../dto/tokenBlueprint.mapper";
import { putFileToSignedUrl } from "../upload/signedUrlPut";

// ---------------------------------------------------------
// API レスポンス型（UI側で使いやすい形）
// ---------------------------------------------------------
export interface TokenBlueprintPageResult {
  items: TokenBlueprint[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
}

// ★ Create 時に iconUpload を発行して欲しい場合のオプション
// NOTE: backend は JSON body の hasIconFile/iconContentType を見ます（header のみでは発行されません）
export type CreateTokenBlueprintOptions = {
  iconFileName?: string;
  iconContentType?: string;
};

// ★ Update 時に iconUpload を発行して欲しい場合のオプション（body で渡す）
export type UpdateTokenBlueprintOptions = {
  hasIconFile?: boolean;
  iconContentType?: string;
};

// ---------------------------------------------------------
// Send payload types (repo-local)
// - dto/tokenBlueprint.dto.ts には存在しないため、ここで定義する
// - entity.go を正として contentFiles は object 配列（ContentFileDTO）
// ---------------------------------------------------------
export type CreateTokenBlueprintPayload = {
  name: string;
  symbol: string;
  brandId: string;
  /** entity.go 正: companyId は必須（互換のため optional 入力は許すが、送信時は必須化） */
  companyId?: string;
  description: string;
  assigneeId: string;
  createdBy: string;
  iconUrl?: string | null;
  contentFiles: ContentFileDTO[];

  // backend create が見るのはこれ（createTokenBlueprintRequest）
  // - create サービス側は options から注入するので通常は UI から触らない
  hasIconFile?: boolean;
  iconContentType?: string;
};

export type UpdateTokenBlueprintPayload = Partial<{
  name: string;
  symbol: string;
  brandId: string;
  /** entity.go 正: companyId は必須だが update では変更不可/不要の運用が多いのでここでは送らない */
  description: string;
  assigneeId: string;
  iconUrl: string | null;
  contentFiles: ContentFileDTO[];
  metadataUri: string;
  minted: boolean;

  // ★ console/handler/updateTokenBlueprintRequest と整合させる
  // update でも iconUpload を返してほしい場合に使う
  hasIconFile: boolean;
  iconContentType: string;
}>;

// ---------------------------------------------------------
// token-contents: signed PUT URL issuance (repo-local)
// ---------------------------------------------------------

export type IssueTokenContentsUploadURLsRequest = {
  files: Array<{
    contentId: string;
    name: string;
    type: "image" | "video" | "pdf" | "document";
    contentType?: string;
    size: number;
    visibility?: "private" | "public";
  }>;
};

// backend の現実に合わせて「upload ネスト」を正とする（互換で flat も optional に残す）
export type IssueTokenContentsUploadURLsResponse = {
  items: Array<{
    contentId: string;
    url: string; // 表示用 URL（cache buster 付き）
    contentFile: any;

    // NEW（推奨）：backend はこれを返す
    upload?: {
      uploadUrl?: string;
      publicUrl?: string;
      objectPath?: string;
      expiresAt?: string;
      contentType?: string;
    };

    // LEGACY（互換）：古い実装が返す可能性があるため optional
    uploadUrl?: string;
    publicUrl?: string;
    objectPath?: string;
  }>;
};

// ---------------------------------------------------------
// Public API
// ---------------------------------------------------------

export async function fetchTokenBlueprints(params?: {
  page?: number;
  perPage?: number;
}): Promise<TokenBlueprintPageResult> {
  const url = new URL("/token-blueprints", "http://local"); // base は後で捨てる
  if (params?.page != null) url.searchParams.set("page", String(params.page));
  if (params?.perPage != null) url.searchParams.set("perPage", String(params.perPage));

  const res = await apiGet(url.pathname + url.search);
  const raw = await handleJsonResponse<any>(res);
  return normalizePageResult(raw);
}

export async function listTokenBlueprintsByCompanyId(companyId: string): Promise<TokenBlueprint[]> {
  const cid = companyId.trim();
  if (!cid) return [];

  // 現状の API が companyId フィルタを持っていない前提で “全件→UI側で絞る” を維持
  const res = await apiGet("/token-blueprints?perPage=200");
  const raw = await handleJsonResponse<any>(res);
  const page = normalizePageResult(raw);

  // companyId 絞り込み（必要なら）
  return page.items.filter((x: any) => String(x?.companyId ?? "").trim() === cid);
}

export async function fetchTokenBlueprintById(id: string): Promise<TokenBlueprint> {
  const trimmed = id.trim();
  if (!trimmed) throw new Error("id is empty");

  const res = await apiGet(`/token-blueprints/${encodeURIComponent(trimmed)}`);
  const raw = await handleJsonResponse<any>(res);
  return normalizeTokenBlueprint(raw);
}

export async function createTokenBlueprint(
  payload: CreateTokenBlueprintPayload,
  options?: CreateTokenBlueprintOptions,
): Promise<TokenBlueprint> {
  const companyId = String(payload.companyId ?? "").trim();
  if (!companyId) {
    // entity.go 正: companyId は必須
    throw new Error("companyId is required");
  }

  const body: any = {
    name: String(payload.name ?? "").trim(),
    symbol: String(payload.symbol ?? "").trim(),
    brandId: String(payload.brandId ?? "").trim(),
    companyId,
    description: String(payload.description ?? "").trim(),
    assigneeId: String(payload.assigneeId ?? "").trim(),
    createdBy: String(payload.createdBy ?? "").trim(),
    iconUrl: payload.iconUrl === undefined ? undefined : payload.iconUrl,
    contentFiles: (payload.contentFiles ?? []).map(normalizeContentFileForSend),
  };

  const headers: Record<string, string> = {};

  const iconCT = String(options?.iconContentType ?? "").trim();
  const iconFN = String(options?.iconFileName ?? "").trim();

  const wantsIconUpload = Boolean(iconCT || iconFN);

  if (wantsIconUpload) {
    // ★重要: backend(createTokenBlueprintRequest) は JSON body を見て iconUpload を返す
    body.hasIconFile = true;
    body.iconContentType = iconCT || "application/octet-stream";

    // ヘッダは「互換/将来用」に残しても害はない（日本語ファイル名は入れない）
    headers["X-Icon-Content-Type"] = body.iconContentType;
    headers["X-Icon-File-Name"] = "icon" + (body.iconContentType === "image/png" ? ".png" : "");
  }

  const res = await apiPostJson("/token-blueprints", body, headers);
  const raw = await handleJsonResponse<any>(res);
  return normalizeTokenBlueprint(raw);
}

export async function updateTokenBlueprint(
  id: string,
  payload: UpdateTokenBlueprintPayload,
  options?: UpdateTokenBlueprintOptions,
): Promise<TokenBlueprint> {
  const trimmed = id.trim();
  if (!trimmed) throw new Error("id is empty");

  const body: UpdateTokenBlueprintPayload = {};

  if (payload.name !== undefined) body.name = payload.name.trim();
  if (payload.symbol !== undefined) body.symbol = payload.symbol.trim();
  if (payload.brandId !== undefined) body.brandId = payload.brandId.trim();
  if (payload.description !== undefined) body.description = payload.description.trim();
  if (payload.assigneeId !== undefined) body.assigneeId = payload.assigneeId.trim();

  if (payload.iconUrl !== undefined) body.iconUrl = payload.iconUrl;

  if (payload.contentFiles !== undefined) {
    body.contentFiles = (payload.contentFiles ?? []).map(normalizeContentFileForSend);
  }

  if (payload.metadataUri !== undefined) body.metadataUri = String(payload.metadataUri).trim();
  if (payload.minted !== undefined) body.minted = !!payload.minted;

  // ★ update で iconUpload を返すためのフラグ/CT（payload と options どちらでも渡せる）
  const hasIconFile =
    typeof payload.hasIconFile === "boolean" ? payload.hasIconFile : Boolean(options?.hasIconFile);

  const iconContentType =
    String(payload.iconContentType ?? "").trim() || String(options?.iconContentType ?? "").trim();

  if (hasIconFile) body.hasIconFile = true;
  if (iconContentType) body.iconContentType = iconContentType;

  const res = await apiPutJson(`/token-blueprints/${encodeURIComponent(trimmed)}`, body);
  const raw = await handleJsonResponse<any>(res);
  return normalizeTokenBlueprint(raw);
}

export async function deleteTokenBlueprint(id: string): Promise<void> {
  const trimmed = id.trim();
  if (!trimmed) throw new Error("id is empty");

  const res = await apiDelete(`/token-blueprints/${encodeURIComponent(trimmed)}`);
  await handleJsonResponse<unknown>(res);
}

// ---------------------------------------------------------
// token-contents helpers (NEW)
// ---------------------------------------------------------

/**
 * POST /token-blueprints/{id}/contents/upload-urls
 * - handler が JSON を返す前提（Cloud Run 側）
 */
export async function issueTokenContentsUploadURLs(params: {
  tokenBlueprintId: string;
  actorId?: string;
  body: IssueTokenContentsUploadURLsRequest;
}): Promise<IssueTokenContentsUploadURLsResponse> {
  const id = params.tokenBlueprintId.trim();
  if (!id) throw new Error("tokenBlueprintId is empty");

  const headers: Record<string, string> = {};
  const actorId = String(params.actorId ?? "").trim();
  if (actorId) headers["X-Actor-Id"] = actorId;

  const res = await apiPostJson(
    `/token-blueprints/${encodeURIComponent(id)}/contents/upload-urls`,
    params.body,
    headers,
  );

  const raw = await handleJsonResponse<any>(res);
  return raw as IssueTokenContentsUploadURLsResponse;
}

/**
 * PATCH /token-blueprints/{id}
 * - contentFiles を差し替える
 */
export async function patchTokenBlueprintContentFiles(params: {
  tokenBlueprintId: string;
  actorId?: string;
  contentFiles: any[];
}): Promise<TokenBlueprint> {
  const id = params.tokenBlueprintId.trim();
  if (!id) throw new Error("tokenBlueprintId is empty");

  const headers: Record<string, string> = {};
  const actorId = String(params.actorId ?? "").trim();
  if (actorId) headers["X-Actor-Id"] = actorId;

  // backend handler は PATCH/PUT の両方を update に通す設計なので、ここでは PUT を採用する
  const res = await apiPutJson(
    `/token-blueprints/${encodeURIComponent(id)}`,
    { contentFiles: params.contentFiles } as any,
    headers as any,
  );

  const raw = await handleJsonResponse<any>(res);
  return normalizeTokenBlueprint(raw);
}

// ---------------------------------------------------------
// Icon helpers
// ---------------------------------------------------------

export async function attachTokenBlueprintIcon(params: {
  tokenBlueprintId: string;
  iconUrl: string;
}): Promise<TokenBlueprint> {
  const id = params.tokenBlueprintId.trim();
  if (!id) throw new Error("tokenBlueprintId is empty");

  const iconUrl = String(params.iconUrl ?? "").trim();
  if (!iconUrl) throw new Error("iconUrl is empty");

  return await updateTokenBlueprint(id, { iconUrl });
}

export async function uploadAndAttachTokenBlueprintIconFromCreateResponse(params: {
  tokenBlueprint: TokenBlueprint;
  file: File;
}): Promise<TokenBlueprint> {
  const tb: any = params.tokenBlueprint as any;
  const file = params.file;
  if (!file) throw new Error("file is empty");

  const id = String(tb?.id ?? "").trim();
  if (!id) throw new Error("tokenBlueprint.id is empty");

  const upl: any = tb?.iconUpload;
  const uploadUrl = String(upl?.uploadUrl ?? "").trim();
  const publicUrl = String(upl?.publicUrl ?? "").trim();
  const contentType =
    upl?.contentType != null ? String(upl.contentType).trim() : undefined;

  if (!uploadUrl || !publicUrl) {
    throw new Error("iconUpload is missing on create response.");
  }

  await putFileToSignedUrl(uploadUrl, file, contentType);

  return await attachTokenBlueprintIcon({
    tokenBlueprintId: id,
    iconUrl: publicUrl,
  });
}

// ---------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------

function normalizeContentFileForSend(x: ContentFileDTO): ContentFileDTO {
  // backend 側 validation を通す前提で、最低限のトリムだけここで行う
  const obj: any = x && typeof x === "object" ? (x as any) : {};

  const visibilityRaw = String(obj.visibility ?? "").trim().toLowerCase();
  const visibility =
    visibilityRaw === "public" || visibilityRaw === "private" ? visibilityRaw : "private";

  const size = Number(obj.size ?? 0);
  const safeSize = Number.isFinite(size) && size > 0 ? size : 0;

  return {
    ...obj,
    id: String(obj.id ?? "").trim(),
    name: String(obj.name ?? "").trim(),
    type: String(obj.type ?? "").trim(),
    contentType: String(obj.contentType ?? "").trim(),
    objectPath: String(obj.objectPath ?? "").trim(),
    visibility,
    size: safeSize,
    createdBy: obj.createdBy != null ? String(obj.createdBy).trim() : obj.createdBy,
    updatedBy: obj.updatedBy != null ? String(obj.updatedBy).trim() : obj.updatedBy,

    // optional timestamps (if DTO includes them)
    createdAt: obj.createdAt != null ? String(obj.createdAt) : obj.createdAt,
    updatedAt: obj.updatedAt != null ? String(obj.updatedAt) : obj.updatedAt,
  } as ContentFileDTO;
}
