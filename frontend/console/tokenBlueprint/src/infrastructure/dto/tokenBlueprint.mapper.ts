// frontend/console/tokenBlueprint/src/infrastructure/dto/tokenBlueprint.mapper.ts

import type {
  TokenBlueprint,
  ContentFile,
} from "../../domain/tokenBlueprint";

import type {
  TokenBlueprintDTO,
  ContentFileDTO,
  ContentFileTypeDTO,
  ContentVisibilityDTO,
} from "./tokenBlueprint.dto";

type RawRecord = Record<string, unknown>;

function asRecord(value: unknown): RawRecord {
  return value && typeof value === "object" ? (value as RawRecord) : {};
}

function toStringValue(value: unknown, fallback = ""): string {
  if (value == null) return fallback;
  return String(value);
}

function toNullableStringValue(value: unknown): string | null {
  if (value == null) return null;

  const s = String(value);
  return s || null;
}

function toBooleanValue(value: unknown, fallback = false): boolean {
  if (typeof value === "boolean") return value;

  if (typeof value === "string") {
    const normalized = value.toLowerCase();
    if (normalized === "true") return true;
    if (normalized === "false") return false;
  }

  return fallback;
}

function toNumberValue(value: unknown, fallback = 0): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const n = Number(value);
    if (Number.isFinite(n)) return n;
  }

  return fallback;
}

function normalizeContentFileType(value: unknown): ContentFileTypeDTO {
  const raw = toStringValue(value).toLowerCase();

  if (
    raw === "image" ||
    raw === "video" ||
    raw === "pdf" ||
    raw === "document"
  ) {
    return raw;
  }

  return "document";
}

function normalizeContentVisibility(value: unknown): ContentVisibilityDTO {
  const raw = toStringValue(value).toLowerCase();

  if (raw === "public" || raw === "private") {
    return raw;
  }

  return "private";
}

function normalizeDateString(value: unknown): string {
  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? "" : value.toISOString();
  }

  return toStringValue(value);
}

/**
 * API レスポンスの contentFiles を domain の ContentFile に変換する。
 *
 * backend 正仕様:
 * - url: Firebase Storage downloadURL
 * - objectPath: Firebase Storage object path
 * - name: 元ファイル名 / 表示名
 * - size: byte size
 * - createdAt / updatedAt: ISO string
 *
 * frontend 表示・差し替え・削除で必要な id / url / objectPath が存在するものだけを採用する。
 */
function normalizeContentFile(raw: unknown): ContentFile | null {
  const obj = asRecord(raw);

  const id = toStringValue(obj.id);
  const name = toStringValue(obj.name);
  const type = normalizeContentFileType(obj.type);
  const contentType =
    toStringValue(obj.contentType) || "application/octet-stream";
  const url = toStringValue(obj.url);
  const objectPath = toStringValue(obj.objectPath);
  const visibility = normalizeContentVisibility(obj.visibility);
  const size = toNumberValue(obj.size, 0);

  const createdAt = normalizeDateString(obj.createdAt);
  const createdBy = toStringValue(obj.createdBy);
  const updatedAt = normalizeDateString(obj.updatedAt);
  const updatedBy = toStringValue(obj.updatedBy);

  if (!id || !url || !objectPath) {
    return null;
  }

  return {
    id,
    name,
    type,
    contentType,
    url,
    objectPath,
    visibility,
    size: Number.isFinite(size) && size >= 0 ? size : 0,
    createdAt,
    createdBy,
    updatedAt,
    updatedBy,
  };
}

function normalizeContentFiles(contentFiles: unknown): ContentFile[] {
  if (!Array.isArray(contentFiles)) {
    return [];
  }

  return contentFiles
    .map((file) => normalizeContentFile(file))
    .filter((file): file is ContentFile => file !== null);
}

export function normalizeTokenBlueprint(raw: unknown): TokenBlueprint {
  const obj = asRecord(raw);

  const id = toStringValue(obj.id);
  const name = toStringValue(obj.name);
  const symbol = toStringValue(obj.symbol);

  const brandId = toStringValue(obj.brandId);
  const brandName = toStringValue(obj.brandName);
  const companyId = toStringValue(obj.companyId);

  const description = toStringValue(obj.description);

  const iconUrl = toNullableStringValue(obj.iconUrl);
  const iconObjectPath = toNullableStringValue(obj.iconObjectPath);
  const iconFileName = toNullableStringValue(obj.iconFileName);
  const iconContentType = toNullableStringValue(obj.iconContentType);
  const iconSizeRaw = obj.iconSize;
  const iconSize = iconSizeRaw == null ? null : toNumberValue(iconSizeRaw, 0);

  const contentFiles = normalizeContentFiles(
    obj.contentFiles as ContentFileDTO[] | undefined,
  );

  const assigneeId = toStringValue(obj.assigneeId);
  const assigneeName = toStringValue(obj.assigneeName);

  const minted = toBooleanValue(obj.minted);

  const createdAt = normalizeDateString(obj.createdAt);
  const createdBy = toStringValue(obj.createdBy);
  const createdByName = toStringValue(obj.createdByName);

  const updatedAt = normalizeDateString(obj.updatedAt);
  const updatedBy = toStringValue(obj.updatedBy);
  const updatedByName = toStringValue(obj.updatedByName);

  const deletedAt = toNullableStringValue(obj.deletedAt);
  const deletedBy = toNullableStringValue(obj.deletedBy);

  const metadataUri = toStringValue(obj.metadataUri);

  return {
    id,
    name,
    symbol,

    brandId,
    brandName,
    companyId,

    description,

    iconUrl,
    iconObjectPath,
    iconFileName,
    iconContentType,
    iconSize,

    minted,

    contentFiles,

    assigneeId,
    assigneeName,

    createdAt,
    createdBy,
    createdByName,

    updatedAt,
    updatedBy,
    updatedByName,

    deletedAt,
    deletedBy,

    metadataUri,
  };
}

export function normalizePageResult(raw: unknown): {
  items: TokenBlueprint[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
} {
  const obj = asRecord(raw);

  const itemsRaw = obj.items as TokenBlueprintDTO[] | undefined;

  return {
    items: Array.isArray(itemsRaw)
      ? itemsRaw.map((item) => normalizeTokenBlueprint(item))
      : [],
    totalCount: toNumberValue(obj.totalCount, 0),
    totalPages: toNumberValue(obj.totalPages, 0),
    page: toNumberValue(obj.page, 1),
    perPage: toNumberValue(obj.perPage, 50),
  };
}