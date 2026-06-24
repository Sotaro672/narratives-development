// frontend/console/tokenBlueprint/src/domain/entity/tokenBlueprint.ts

/**
 * TokenBlueprint domain entity
 *
 * backend/internal/domain/tokenBlueprint/entity.go と一致させる。
 *
 * Firebase Storage 移行後の正仕様:
 * - frontend が Firebase Storage へ直接 upload する
 * - backend は signed URL / upload endpoint を持たない
 * - url は Firebase Storage downloadURL
 * - objectPath は Firebase Storage 上の実体を差し替え・削除するための正規キー
 * - icon / contentFiles ともに objectPath を保存する
 */

/* =========================================================
 * Content type / visibility
 * =======================================================*/

/**
 * backend ContentFileType に対応。
 *
 * backend:
 * type ContentFileType string
 * const:
 * - image
 * - video
 * - pdf
 * - document
 */
export type ContentType = "image" | "video" | "pdf" | "document";

export type ContentFileType = ContentType;

export const ALL_CONTENT_TYPES: ContentType[] = [
  "image",
  "video",
  "pdf",
  "document",
];

export function isValidContentType(value: string): value is ContentType {
  return (
    value === "image" ||
    value === "video" ||
    value === "pdf" ||
    value === "document"
  );
}

/**
 * backend ContentVisibility に対応。
 *
 * backend:
 * type ContentVisibility string
 * const:
 * - private
 * - public
 */
export type ContentVisibility = "private" | "public";

export const ALL_CONTENT_VISIBILITIES: ContentVisibility[] = [
  "private",
  "public",
];

export function isValidContentVisibility(
  value: string,
): value is ContentVisibility {
  return value === "private" || value === "public";
}

/* =========================================================
 * ContentFile
 * =======================================================*/

/**
 * ContentFile
 *
 * backend/internal/domain/tokenBlueprint/entity.go の ContentFile と一致。
 *
 * backend:
 * type ContentFile struct {
 *   ID          string            `json:"id"`
 *   Name        string            `json:"name"`
 *   Type        ContentFileType   `json:"type"`
 *   ContentType string            `json:"contentType,omitempty"`
 *   URL         string            `json:"url"`
 *   ObjectPath  string            `json:"objectPath"`
 *   Visibility  ContentVisibility `json:"visibility"`
 *   Size        int64             `json:"size"`
 *   CreatedAt   time.Time         `json:"createdAt"`
 *   CreatedBy   string            `json:"createdBy"`
 *   UpdatedAt   time.Time         `json:"updatedAt"`
 *   UpdatedBy   string            `json:"updatedBy"`
 * }
 */
export interface ContentFile {
  id: string;
  name: string;
  type: ContentType;
  contentType: string;
  url: string;
  objectPath: string;
  visibility: ContentVisibility;
  size: number;

  createdAt: string;
  createdBy: string;
  updatedAt: string;
  updatedBy: string;
}

/**
 * Token content 表示用。
 *
 * 現在は backend ContentFile と同一形で扱う。
 * 旧 tokenContents.ts の FirebaseStorageTokenContent はこの型に統合する。
 */
export type FirebaseStorageTokenContent = ContentFile;

/* =========================================================
 * TokenIcon
 * =======================================================*/

/**
 * TokenIcon
 *
 * backend TokenBlueprint の icon 系 field と対応。
 *
 * backend:
 * - IconURL
 * - IconObjectPath
 * - IconFileName
 * - IconContentType
 * - IconSize
 */
export interface TokenIcon {
  id: string;
  url: string;
  objectPath: string;
  fileName: string;
  contentType: string;
  size: number;
}

/* =========================================================
 * TokenBlueprint
 * =======================================================*/

/**
 * TokenBlueprint
 *
 * backend/internal/domain/tokenBlueprint/entity.go の TokenBlueprint と一致。
 *
 * backend:
 * type TokenBlueprint struct {
 *   ID              string
 *   Name            string
 *   Symbol          string
 *   BrandID         string
 *   CompanyID       string
 *   Description     string
 *   IconURL         string
 *   IconObjectPath  string
 *   IconFileName    string
 *   IconContentType string
 *   IconSize        int64
 *   ContentFiles    []ContentFile
 *   AssigneeID      string
 *   Minted          bool
 *   CreatedAt       time.Time
 *   CreatedBy       string
 *   UpdatedAt       time.Time
 *   UpdatedBy       string
 *   DeletedAt       *time.Time
 *   DeletedBy       *string
 *   MetadataURI     string
 * }
 *
 * brandName / assigneeName / createdByName / updatedByName は
 * backend query / resolver が画面表示用に付与する補助 field。
 */
export interface TokenBlueprint {
  id: string;
  name: string;
  symbol: string;

  brandId: string;
  brandName?: string;
  companyId: string;

  description?: string;

  iconUrl?: string | null;
  iconObjectPath?: string | null;
  iconFileName?: string | null;
  iconContentType?: string | null;
  iconSize?: number | null;

  contentFiles: ContentFile[];

  assigneeId: string;
  assigneeName?: string;

  minted: boolean;

  createdAt?: string;
  createdBy?: string;
  createdByName?: string;

  updatedAt?: string;
  updatedBy?: string;
  updatedByName?: string;

  deletedAt?: string | null;
  deletedBy?: string | null;

  metadataUri?: string;
}

/* =========================================================
 * Firebase Storage delete operation
 * =======================================================*/

/**
 * Firebase Storage delete operation.
 *
 * Firebase Storage 実体の削除・差し替えは objectPath を正規キーにする。
 */
export interface FirebaseStorageDeleteOp {
  objectPath: string;
}

export function toContentFileFirebaseStorageDeleteOp(
  content: ContentFile,
): FirebaseStorageDeleteOp {
  return {
    objectPath: content.objectPath,
  };
}

export function toTokenIconFirebaseStorageDeleteOp(
  icon: TokenIcon,
): FirebaseStorageDeleteOp {
  return {
    objectPath: icon.objectPath,
  };
}

/* =========================================================
 * Icon policy
 * =======================================================*/

export const TOKEN_ICON_MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB

export const TOKEN_ICON_ALLOWED_EXTENSIONS = [
  ".png",
  ".jpg",
  ".jpeg",
  ".webp",
  ".gif",
] as const;

export const TOKEN_ICON_ALLOWED_CONTENT_TYPES = [
  "image/png",
  "image/jpeg",
  "image/webp",
  "image/gif",
] as const;

/* =========================================================
 * Content helpers
 * =======================================================*/

export function normalizeContentType(value: unknown): ContentType {
  const raw = String(value ?? "").toLowerCase();

  if (isValidContentType(raw)) {
    return raw;
  }

  return "document";
}

export function normalizeContentVisibility(value: unknown): ContentVisibility {
  const raw = String(value ?? "").toLowerCase();

  if (isValidContentVisibility(raw)) {
    return raw;
  }

  return "private";
}

export function validateContentFile(content: ContentFile): string[] {
  const errors: string[] = [];

  if (!content.id) {
    errors.push("id is required");
  }

  if (!content.name) {
    errors.push("name is required");
  }

  if (!isValidContentType(content.type)) {
    errors.push("type must be one of 'image' | 'video' | 'pdf' | 'document'");
  }

  if (!content.contentType) {
    errors.push("contentType is required");
  }

  if (!content.url) {
    errors.push("url is required");
  } else if (!isValidHttpUrl(content.url)) {
    errors.push("url must be a valid http(s) URL");
  }

  if (!content.objectPath) {
    errors.push("objectPath is required");
  }

  if (!isValidContentVisibility(content.visibility)) {
    errors.push("visibility must be one of 'private' | 'public'");
  }

  if (!Number.isFinite(content.size) || content.size < 0) {
    errors.push("size must be 0 or greater");
  }

  if (!content.createdAt) {
    errors.push("createdAt is required");
  }

  if (!content.createdBy) {
    errors.push("createdBy is required");
  }

  if (!content.updatedAt) {
    errors.push("updatedAt is required");
  }

  if (!content.updatedBy) {
    errors.push("updatedBy is required");
  }

  return errors;
}

export function createContentFile(input: ContentFile): ContentFile {
  const normalized: ContentFile = {
    id: input.id,
    name: input.name,
    type: normalizeContentType(input.type),
    contentType:
      input.contentType || "application/octet-stream",
    url: input.url,
    objectPath: input.objectPath,
    visibility: normalizeContentVisibility(input.visibility),
    size: Number.isFinite(input.size) && input.size >= 0 ? input.size : 0,

    createdAt: input.createdAt,
    createdBy: input.createdBy,
    updatedAt: input.updatedAt,
    updatedBy: input.updatedBy,
  };

  const errors = validateContentFile(normalized);
  if (errors.length > 0) {
    throw new Error(`Invalid ContentFile: ${errors.join(", ")}`);
  }

  return normalized;
}

export function validateContentFiles(contents: ContentFile[]): string[] {
  const errors: string[] = [];
  const ids = new Set<string>();
  const objectPaths = new Set<string>();

  contents.forEach((content, index) => {
    const contentErrors = validateContentFile(content);

    for (const error of contentErrors) {
      errors.push(`contentFiles[${index}].${error}`);
    }

    const id = content.id;
    if (id) {
      if (ids.has(id)) {
        errors.push(`contentFiles[${index}].id duplicated`);
      }
      ids.add(id);
    }

    const objectPath = content.objectPath;
    if (objectPath) {
      if (objectPaths.has(objectPath)) {
        errors.push(`contentFiles[${index}].objectPath duplicated`);
      }
      objectPaths.add(objectPath);
    }
  });

  return errors;
}

/* =========================================================
 * Icon helpers
 * =======================================================*/

export function isTokenIconExtensionAllowed(fileName: string): boolean {
  if (!TOKEN_ICON_ALLOWED_EXTENSIONS.length) return true;

  const lower = fileName.toLowerCase();

  return TOKEN_ICON_ALLOWED_EXTENSIONS.some((ext) => lower.endsWith(ext));
}

export function isTokenIconContentTypeAllowed(contentType: string): boolean {
  const normalized = contentType.toLowerCase();

  return TOKEN_ICON_ALLOWED_CONTENT_TYPES.some(
    (allowed) => allowed === normalized,
  );
}

export function validateTokenIcon(icon: TokenIcon): boolean {
  if (!icon.id) return false;

  if (!icon.url) return false;
  if (!isValidHttpUrl(icon.url)) return false;

  if (!icon.objectPath) return false;

  if (!icon.fileName) return false;

  if (!icon.contentType) return false;
  if (!isTokenIconContentTypeAllowed(icon.contentType)) return false;

  if (!Number.isFinite(icon.size) || icon.size < 0) return false;
  if (TOKEN_ICON_MAX_FILE_SIZE > 0 && icon.size > TOKEN_ICON_MAX_FILE_SIZE) {
    return false;
  }

  return true;
}

export function validateTokenIconFile(file: File): boolean {
  if (!file) return false;

  if (!file.name) return false;
  if (!isTokenIconExtensionAllowed(file.name)) return false;

  if (!file.type) return false;
  if (!isTokenIconContentTypeAllowed(file.type)) return false;

  if (!Number.isFinite(file.size) || file.size < 0) return false;
  if (TOKEN_ICON_MAX_FILE_SIZE > 0 && file.size > TOKEN_ICON_MAX_FILE_SIZE) {
    return false;
  }

  return true;
}

export function createTokenIcon(input: TokenIcon): TokenIcon {
  const normalized: TokenIcon = {
    id: input.id,
    url: input.url,
    objectPath: input.objectPath,
    fileName: input.fileName,
    contentType: input.contentType,
    size: Number.isFinite(input.size) && input.size >= 0 ? input.size : 0,
  };

  if (!validateTokenIcon(normalized)) {
    throw new Error("Invalid TokenIcon");
  }

  return normalized;
}

/* =========================================================
 * TokenBlueprint helpers
 * =======================================================*/

export function validateTokenBlueprint(input: TokenBlueprint): string[] {
  const errors: string[] = [];

  if (!input.id) {
    errors.push("id is required");
  }

  if (!input.name) {
    errors.push("name is required");
  }

  if (!input.symbol) {
    errors.push("symbol is required");
  }

  if (!input.brandId) {
    errors.push("brandId is required");
  }

  if (!input.companyId) {
    errors.push("companyId is required");
  }

  if (!input.assigneeId) {
    errors.push("assigneeId is required");
  }

  const hasAnyIconField =
    Boolean(input.iconUrl) ||
    Boolean(input.iconObjectPath) ||
    Boolean(input.iconFileName) ||
    Boolean(input.iconContentType) ||
    input.iconSize != null;

  if (hasAnyIconField) {
    if (!input.iconUrl) {
      errors.push("iconUrl is required when icon is set");
    }

    if (input.iconUrl && !isValidHttpUrl(input.iconUrl)) {
      errors.push("iconUrl must be a valid http(s) URL");
    }

    if (!input.iconObjectPath) {
      errors.push("iconObjectPath is required when icon is set");
    }

    if (!input.iconFileName) {
      errors.push("iconFileName is required when icon is set");
    }

    if (input.iconSize != null && (!Number.isFinite(input.iconSize) || input.iconSize < 0)) {
      errors.push("iconSize must be 0 or greater");
    }
  }

  errors.push(...validateContentFiles(input.contentFiles ?? []));

  return errors;
}

/* =========================================================
 * internal helpers
 * =======================================================*/

function isValidHttpUrl(raw: string): boolean {
  try {
    const u = new URL(raw);
    if (!u.protocol || !u.hostname) return false;
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}