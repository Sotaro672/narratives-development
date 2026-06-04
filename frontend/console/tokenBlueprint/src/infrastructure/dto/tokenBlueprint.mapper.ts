// frontend/console/tokenBlueprint/src/infrastructure/dto/tokenBlueprint.mapper.ts

import type {
  TokenBlueprint,
  ContentFile,
} from "../../domain/entity/tokenBlueprint";

import type {
  TokenBlueprintDTO,
  TokenBlueprintPageResultDTO,
  ContentFileDTO,
  ContentFileTypeDTO,
  ContentVisibilityDTO,
} from "./tokenBlueprint.dto";

type RawRecord = Record<string, unknown>;

function asRecord(value: unknown): RawRecord {
  return value && typeof value === "object" ? (value as RawRecord) : {};
}

/**
 * backend response の camelCase / PascalCase 揺れをここで1回だけ吸収する。
 *
 * frontend domain / presentation へ渡す値は camelCase に統一する。
 */
function pick<T = unknown>(
  obj: RawRecord,
  camelKey: string,
  pascalKey?: string,
): T | undefined {
  const pascal = pascalKey ?? camelKey.charAt(0).toUpperCase() + camelKey.slice(1);

  if (obj[camelKey] !== undefined) {
    return obj[camelKey] as T;
  }

  if (obj[pascal] !== undefined) {
    return obj[pascal] as T;
  }

  return undefined;
}

function toStringValue(value: unknown, fallback = ""): string {
  if (value == null) return fallback;
  return String(value).trim();
}

function toNullableStringValue(value: unknown): string | null {
  if (value == null) return null;

  const s = String(value).trim();
  return s || null;
}

function toBooleanValue(value: unknown, fallback = false): boolean {
  if (typeof value === "boolean") return value;

  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
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

  const id = toStringValue(pick(obj, "id", "ID"));
  const name = toStringValue(pick(obj, "name", "Name"));
  const type = normalizeContentFileType(pick(obj, "type", "Type"));
  const contentType =
    toStringValue(pick(obj, "contentType", "ContentType")) ||
    "application/octet-stream";
  const url = toStringValue(pick(obj, "url", "URL"));
  const objectPath = toStringValue(pick(obj, "objectPath", "ObjectPath"));
  const visibility = normalizeContentVisibility(
    pick(obj, "visibility", "Visibility"),
  );
  const size = toNumberValue(pick(obj, "size", "Size"), 0);

  const createdAt = normalizeDateString(pick(obj, "createdAt", "CreatedAt"));
  const createdBy = toStringValue(pick(obj, "createdBy", "CreatedBy"));
  const updatedAt = normalizeDateString(pick(obj, "updatedAt", "UpdatedAt"));
  const updatedBy = toStringValue(pick(obj, "updatedBy", "UpdatedBy"));

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

  const id = toStringValue(pick(obj, "id", "ID"));
  const name = toStringValue(pick(obj, "name", "Name"));
  const symbol = toStringValue(pick(obj, "symbol", "Symbol"));

  const brandId = toStringValue(pick(obj, "brandId", "BrandID"));
  const brandName = toStringValue(pick(obj, "brandName", "BrandName"));
  const companyId = toStringValue(pick(obj, "companyId", "CompanyID"));

  const description = toStringValue(pick(obj, "description", "Description"));

  const iconUrl = toNullableStringValue(pick(obj, "iconUrl", "IconURL"));
  const iconObjectPath = toNullableStringValue(
    pick(obj, "iconObjectPath", "IconObjectPath"),
  );
  const iconFileName = toNullableStringValue(
    pick(obj, "iconFileName", "IconFileName"),
  );
  const iconContentType = toNullableStringValue(
    pick(obj, "iconContentType", "IconContentType"),
  );
  const iconSizeRaw = pick(obj, "iconSize", "IconSize");
  const iconSize =
    iconSizeRaw == null ? null : toNumberValue(iconSizeRaw, 0);

  const contentFiles = normalizeContentFiles(
    pick<ContentFileDTO[]>(obj, "contentFiles", "ContentFiles"),
  );

  const assigneeId = toStringValue(pick(obj, "assigneeId", "AssigneeID"));
  const assigneeName = toStringValue(
    pick(obj, "assigneeName", "AssigneeName"),
  );

  const minted = toBooleanValue(pick(obj, "minted", "Minted"));

  const createdAt = normalizeDateString(pick(obj, "createdAt", "CreatedAt"));
  const createdBy = toStringValue(pick(obj, "createdBy", "CreatedBy"));
  const createdByName = toStringValue(
    pick(obj, "createdByName", "CreatedByName"),
  );

  const updatedAt = normalizeDateString(pick(obj, "updatedAt", "UpdatedAt"));
  const updatedBy = toStringValue(pick(obj, "updatedBy", "UpdatedBy"));
  const updatedByName = toStringValue(
    pick(obj, "updatedByName", "UpdatedByName"),
  );

  const deletedAt = toNullableStringValue(pick(obj, "deletedAt", "DeletedAt"));
  const deletedBy = toNullableStringValue(pick(obj, "deletedBy", "DeletedBy"));

  const metadataUri = toStringValue(pick(obj, "metadataUri", "MetadataURI"));

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

  const itemsRaw =
    pick<TokenBlueprintDTO[]>(obj, "items", "Items") ??
    [];

  return {
    items: Array.isArray(itemsRaw)
      ? itemsRaw.map((item) => normalizeTokenBlueprint(item))
      : [],
    totalCount: toNumberValue(pick(obj, "totalCount", "TotalCount"), 0),
    totalPages: toNumberValue(pick(obj, "totalPages", "TotalPages"), 0),
    page: toNumberValue(pick(obj, "page", "Page"), 1),
    perPage: toNumberValue(pick(obj, "perPage", "PerPage"), 50),
  };
}