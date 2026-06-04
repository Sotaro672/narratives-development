// frontend/console/tokenBlueprint/src/infrastructure/repository/tokenBlueprintRepositoryHTTP.ts

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";
import type { ContentFileDTO } from "../dto/tokenBlueprint.dto";

import { handleJsonResponse } from "../http/json";
import { apiDelete, apiGet, apiPostJson, apiPutJson } from "../http/client";
import {
  normalizePageResult,
  normalizeTokenBlueprint,
} from "../dto/tokenBlueprint.mapper";

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

// ---------------------------------------------------------
// Send payload types
// - tokenBlueprintIcon / tokenBlueprintContents は Firebase Storage に
//   frontend から直接アップロードする
// - iconUrl は Firebase Storage downloadURL
// - iconObjectPath は Firebase Storage objectPath
// - contentFiles[].url は Firebase Storage downloadURL
// - contentFiles[].objectPath は Firebase Storage objectPath
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
  iconObjectPath?: string | null;
  iconFileName?: string | null;
  iconContentType?: string | null;
  iconSize?: number | null;

  contentFiles: ContentFileDTO[];
};

export type UpdateTokenBlueprintPayload = Partial<{
  name: string;
  symbol: string;
  description: string;
  assigneeId: string;

  iconUrl: string | null;
  iconObjectPath: string | null;
  iconFileName: string | null;
  iconContentType: string | null;
  iconSize: number | null;

  contentFiles: ContentFileDTO[];
}>;

// ---------------------------------------------------------
// Public API
// ---------------------------------------------------------

export async function fetchTokenBlueprints(params?: {
  page?: number;
  perPage?: number;
}): Promise<TokenBlueprintPageResult> {
  const url = new URL("/token-blueprints", "http://local");

  if (params?.page != null) {
    url.searchParams.set("page", String(params.page));
  }

  if (params?.perPage != null) {
    url.searchParams.set("perPage", String(params.perPage));
  }

  const res = await apiGet(url.pathname + url.search);
  const raw = await handleJsonResponse<unknown>(res);

  return normalizePageResult(raw);
}

/**
 * 互換用。
 *
 * backend 側で CompanyIDFromContext(ctx) により company boundary を張るため、
 * 通常は fetchTokenBlueprints を使う。
 */
export async function listTokenBlueprintsByCompanyId(
  companyId: string,
): Promise<TokenBlueprint[]> {
  const cid = companyId.trim();
  if (!cid) return [];

  const res = await apiGet("/token-blueprints?perPage=200");
  const raw = await handleJsonResponse<unknown>(res);
  const page = normalizePageResult(raw);

  return page.items.filter((x) => {
    return String(x.companyId ?? "").trim() === cid;
  });
}

export async function fetchTokenBlueprintById(
  id: string,
): Promise<TokenBlueprint> {
  const trimmed = id.trim();
  if (!trimmed) throw new Error("id is empty");

  const res = await apiGet(`/token-blueprints/${encodeURIComponent(trimmed)}`);
  const raw = await handleJsonResponse<unknown>(res);

  return normalizeTokenBlueprint(raw);
}

export async function createTokenBlueprint(
  payload: CreateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  const companyId = String(payload.companyId ?? "").trim();
  if (!companyId) {
    throw new Error("companyId is required");
  }

  const body: CreateTokenBlueprintPayload = {
    name: String(payload.name ?? "").trim(),
    symbol: String(payload.symbol ?? "").trim(),
    brandId: String(payload.brandId ?? "").trim(),
    companyId,
    description: String(payload.description ?? "").trim(),
    assigneeId: String(payload.assigneeId ?? "").trim(),
    createdBy: String(payload.createdBy ?? "").trim(),

    iconUrl: normalizeOptionalString(payload.iconUrl),
    iconObjectPath: normalizeOptionalString(payload.iconObjectPath),
    iconFileName: normalizeOptionalString(payload.iconFileName),
    iconContentType: normalizeOptionalString(payload.iconContentType),
    iconSize: normalizeOptionalNumber(payload.iconSize),

    contentFiles: (payload.contentFiles ?? []).map(normalizeContentFileForSend),
  };

  const res = await apiPostJson("/token-blueprints", body);
  const raw = await handleJsonResponse<unknown>(res);

  return normalizeTokenBlueprint(raw);
}

export async function updateTokenBlueprint(
  id: string,
  payload: UpdateTokenBlueprintPayload,
): Promise<TokenBlueprint> {
  const trimmed = id.trim();
  if (!trimmed) throw new Error("id is empty");

  const body: UpdateTokenBlueprintPayload = {};

  if (payload.name !== undefined) {
    body.name = payload.name.trim();
  }

  if (payload.symbol !== undefined) {
    body.symbol = payload.symbol.trim();
  }

  if (payload.description !== undefined) {
    body.description = payload.description.trim();
  }

  if (payload.assigneeId !== undefined) {
    body.assigneeId = payload.assigneeId.trim();
  }

  if (payload.iconUrl !== undefined) {
    body.iconUrl = normalizeNullableString(payload.iconUrl);
  }

  if (payload.iconObjectPath !== undefined) {
    body.iconObjectPath = normalizeNullableString(payload.iconObjectPath);
  }

  if (payload.iconFileName !== undefined) {
    body.iconFileName = normalizeNullableString(payload.iconFileName);
  }

  if (payload.iconContentType !== undefined) {
    body.iconContentType = normalizeNullableString(payload.iconContentType);
  }

  if (payload.iconSize !== undefined) {
    body.iconSize = normalizeNullableNumber(payload.iconSize);
  }

  if (payload.contentFiles !== undefined) {
    body.contentFiles = (payload.contentFiles ?? []).map(
      normalizeContentFileForSend,
    );
  }

  const res = await apiPutJson(
    `/token-blueprints/${encodeURIComponent(trimmed)}`,
    body,
  );
  const raw = await handleJsonResponse<unknown>(res);

  return normalizeTokenBlueprint(raw);
}

export async function deleteTokenBlueprint(id: string): Promise<void> {
  const trimmed = id.trim();
  if (!trimmed) throw new Error("id is empty");

  const res = await apiDelete(
    `/token-blueprints/${encodeURIComponent(trimmed)}`,
  );

  await handleJsonResponse<unknown>(res);
}

// ---------------------------------------------------------
// token-contents helpers
// ---------------------------------------------------------

/**
 * PUT /token-blueprints/{id}
 * - Firebase Storage へ frontend から直接 upload した後、
 *   downloadURL / objectPath を含む contentFiles を backend に保存する。
 */
export async function patchTokenBlueprintContentFiles(params: {
  tokenBlueprintId: string;
  contentFiles: ContentFileDTO[];
}): Promise<TokenBlueprint> {
  const id = params.tokenBlueprintId.trim();
  if (!id) throw new Error("tokenBlueprintId is empty");

  const contentFiles = (params.contentFiles ?? []).map(
    normalizeContentFileForSend,
  );

  const res = await apiPutJson(
    `/token-blueprints/${encodeURIComponent(id)}`,
    { contentFiles },
  );

  const raw = await handleJsonResponse<unknown>(res);

  return normalizeTokenBlueprint(raw);
}

// ---------------------------------------------------------
// Icon helpers
// ---------------------------------------------------------

export async function attachTokenBlueprintIcon(params: {
  tokenBlueprintId: string;
  iconUrl: string;
  iconObjectPath: string;
  iconFileName?: string | null;
  iconContentType?: string | null;
  iconSize?: number | null;
}): Promise<TokenBlueprint> {
  const id = params.tokenBlueprintId.trim();
  if (!id) throw new Error("tokenBlueprintId is empty");

  const iconUrl = String(params.iconUrl ?? "").trim();
  if (!iconUrl) throw new Error("iconUrl is empty");

  const iconObjectPath = String(params.iconObjectPath ?? "").trim();
  if (!iconObjectPath) throw new Error("iconObjectPath is empty");

  return await updateTokenBlueprint(id, {
    iconUrl,
    iconObjectPath,
    iconFileName: params.iconFileName ?? null,
    iconContentType: params.iconContentType ?? null,
    iconSize: params.iconSize ?? null,
  });
}

// ---------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------

function normalizeOptionalString(value: unknown): string | undefined {
  if (value === undefined) return undefined;
  if (value === null) return undefined;

  return String(value).trim();
}

function normalizeNullableString(value: unknown): string | null {
  if (value === undefined) return null;
  if (value === null) return null;

  const s = String(value).trim();
  return s || null;
}

function normalizeOptionalNumber(value: unknown): number | undefined {
  if (value === undefined || value === null) return undefined;

  const n = Number(value);
  if (!Number.isFinite(n)) return undefined;

  return n >= 0 ? n : 0;
}

function normalizeNullableNumber(value: unknown): number | null {
  if (value === undefined || value === null) return null;

  const n = Number(value);
  if (!Number.isFinite(n)) return null;

  return n >= 0 ? n : 0;
}

function toIsoStringOrNow(value: unknown): string {
  if (value instanceof Date) {
    if (Number.isNaN(value.getTime())) {
      return new Date().toISOString();
    }

    return value.toISOString();
  }

  const raw = String(value ?? "").trim();
  if (!raw) {
    return new Date().toISOString();
  }

  const parsed = new Date(raw);
  if (Number.isNaN(parsed.getTime())) {
    return new Date().toISOString();
  }

  return parsed.toISOString();
}

function normalizeContentFileType(value: unknown): ContentFileDTO["type"] {
  const raw = String(value ?? "").trim().toLowerCase();

  if (
    raw === "image" ||
    raw === "video" ||
    raw === "pdf" ||
    raw === "document"
  ) {
    return raw as ContentFileDTO["type"];
  }

  return "document" as ContentFileDTO["type"];
}

function normalizeContentVisibility(
  value: unknown,
): ContentFileDTO["visibility"] {
  const raw = String(value ?? "").trim().toLowerCase();

  if (raw === "public" || raw === "private") {
    return raw as ContentFileDTO["visibility"];
  }

  return "private" as ContentFileDTO["visibility"];
}

function normalizeContentFileForSend(x: ContentFileDTO): ContentFileDTO {
  const obj: any = x && typeof x === "object" ? (x as any) : {};

  const id = String(obj.id ?? "").trim();
  const name = String(obj.name ?? "").trim();
  const type = normalizeContentFileType(obj.type);
  const contentType =
    String(obj.contentType ?? "").trim() || "application/octet-stream";
  const objectPath = String(obj.objectPath ?? "").trim();
  const url = String(obj.url ?? "").trim();
  const visibility = normalizeContentVisibility(obj.visibility);

  const size = Number(obj.size ?? 0);
  const safeSize = Number.isFinite(size) && size >= 0 ? size : 0;

  const nowIso = new Date().toISOString();

  const createdBy =
    obj.createdBy != null ? String(obj.createdBy).trim() : "";
  const updatedBy =
    obj.updatedBy != null ? String(obj.updatedBy).trim() : "";

  if (!id) throw new Error("contentFile.id is required");
  if (!name) throw new Error("contentFile.name is required");
  if (!url) throw new Error("contentFile.url is required");
  if (!objectPath) throw new Error("contentFile.objectPath is required");
  if (!createdBy) throw new Error("contentFile.createdBy is required");
  if (!updatedBy) throw new Error("contentFile.updatedBy is required");

  return {
    id,
    name,
    type,
    contentType,
    objectPath,
    url,
    visibility,
    size: safeSize,
    createdBy,
    updatedBy,
    createdAt: toIsoStringOrNow(obj.createdAt || nowIso),
    updatedAt: toIsoStringOrNow(obj.updatedAt || nowIso),
  };
}