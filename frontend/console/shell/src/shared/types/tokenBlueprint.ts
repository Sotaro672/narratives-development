// frontend/console/shell/src/shared/types/tokenBlueprint.ts

/**
 * TokenBlueprint共通型・純粋関数・ファイルポリシー。
 *
 * このファイルをFrontendにおけるTokenBlueprint型の
 * 唯一の正規定義とする。
 *
 * Feature配下で同名の型を再定義せず、
 * 必要な型・定数・純粋関数はこのファイルから
 * importまたはre-exportする。
 *
 * 対象:
 * - TokenBlueprint
 * - TokenIcon
 * - ContentFile
 * - ContentType
 * - ContentFile.isPublic
 * - TokenBlueprintコンテンツポリシー
 * - TokenBlueprintアイコンポリシー
 * - 純粋な正規化・検証関数
 *
 * 対象外:
 * - HTTP DTO
 * - Firebase Storage操作
 * - API通信
 * - Reactの状態
 * - ViewModel
 * - Application Service
 */

/* =========================================================
 * Content type
 * =======================================================*/

export type ContentType =
  | "image"
  | "video"
  | "pdf"
  | "document";

export type ContentFileType = ContentType;

export type TokenBlueprintContentKind =
  ContentType;

export const ALL_CONTENT_TYPES: readonly ContentType[] = [
  "image",
  "video",
  "pdf",
  "document",
];

export function isValidContentType(
  value: string,
): value is ContentType {
  return (
    value === "image" ||
    value === "video" ||
    value === "pdf" ||
    value === "document"
  );
}

export function normalizeContentType(
  value: unknown,
): ContentType {
  const normalized = String(
    value ?? "",
  ).toLowerCase();

  if (isValidContentType(normalized)) {
    return normalized;
  }

  return "document";
}

/* =========================================================
 * Content publication state
 * =======================================================*/

/**
 * ContentFile.isPublicをbooleanへ正規化する。
 *
 * 正仕様:
 * - true: 公開
 * - false: 非公開
 *
 * 旧仕様の"public" / "private"は受け付けない。
 */
export function normalizeContentIsPublic(
  value: unknown,
  fallback = false,
): boolean {
  if (typeof value === "boolean") {
    return value;
  }

  if (typeof value === "string") {
    const normalized =
      value.trim().toLowerCase();

    if (normalized === "true") {
      return true;
    }

    if (normalized === "false") {
      return false;
    }
  }

  return fallback;
}

/* =========================================================
 * TokenBlueprint content policy
 * =======================================================*/

export const TOKEN_BLUEPRINT_DEFAULT_CONTENT_TYPE =
  "application/octet-stream";

/**
 * MIMEタイプを正規化する。
 *
 * 未設定の場合はapplication/octet-streamを返す。
 */
export function normalizeTokenBlueprintMimeType(
  value: unknown,
): string {
  const normalized = String(
    value ?? "",
  ).trim();

  return (
    normalized ||
    TOKEN_BLUEPRINT_DEFAULT_CONTENT_TYPE
  );
}

/**
 * MIMEタイプからTokenBlueprintのコンテンツ種別を判定する。
 *
 * 対応:
 * - image/* → image
 * - video/* → video
 * - application/pdf → pdf
 * - その他 → document
 */
export function getTokenBlueprintContentTypeFromMimeType(
  value: unknown,
): TokenBlueprintContentKind {
  const mimeType =
    normalizeTokenBlueprintMimeType(
      value,
    ).toLowerCase();

  if (mimeType.startsWith("image/")) {
    return "image";
  }

  if (mimeType.startsWith("video/")) {
    return "video";
  }

  if (mimeType === "application/pdf") {
    return "pdf";
  }

  return "document";
}

/**
 * FileオブジェクトからTokenBlueprintの
 * コンテンツ種別を判定する。
 */
export function guessTokenBlueprintContentType(
  file: Pick<File, "type">,
): TokenBlueprintContentKind {
  return getTokenBlueprintContentTypeFromMimeType(
    file.type,
  );
}

/* =========================================================
 * ContentFile
 * =======================================================*/

export interface ContentFile {
  id: string;
  name: string;

  type: ContentType;
  contentType: string;

  url: string;
  objectPath: string;

  isPublic: boolean;
  size: number;

  createdAt: string;
  createdBy: string;

  updatedAt: string;
  updatedBy: string;
}

/**
 * TokenContentsCard表示用の旧名称。
 *
 * 現在はContentFileと同じ構造であるため、
 * 別のinterfaceは作らずaliasとして扱う。
 */
export type FirebaseStorageTokenContent =
  ContentFile;

export function validateContentFile(
  content: ContentFile,
): string[] {
  const errors: string[] = [];

  if (!content.id) {
    errors.push("id is required");
  }

  if (!content.name) {
    errors.push("name is required");
  }

  if (
    !isValidContentType(
      content.type,
    )
  ) {
    errors.push(
      "type must be one of 'image' | 'video' | 'pdf' | 'document'",
    );
  }

  if (!content.contentType) {
    errors.push(
      "contentType is required",
    );
  }

  if (!content.url) {
    errors.push("url is required");
  } else if (
    !isValidHttpUrl(content.url)
  ) {
    errors.push(
      "url must be a valid http(s) URL",
    );
  }

  if (!content.objectPath) {
    errors.push(
      "objectPath is required",
    );
  }

  if (
    typeof content.isPublic !==
    "boolean"
  ) {
    errors.push(
      "isPublic must be boolean",
    );
  }

  if (
    !Number.isFinite(content.size) ||
    content.size < 0
  ) {
    errors.push(
      "size must be 0 or greater",
    );
  }

  if (!content.createdAt) {
    errors.push(
      "createdAt is required",
    );
  }

  if (!content.createdBy) {
    errors.push(
      "createdBy is required",
    );
  }

  if (!content.updatedAt) {
    errors.push(
      "updatedAt is required",
    );
  }

  if (!content.updatedBy) {
    errors.push(
      "updatedBy is required",
    );
  }

  return errors;
}

export function createContentFile(
  input: ContentFile,
): ContentFile {
  const normalized: ContentFile = {
    id: input.id,
    name: input.name,

    type: normalizeContentType(
      input.type,
    ),

    contentType:
      normalizeTokenBlueprintMimeType(
        input.contentType,
      ),

    url: input.url,
    objectPath: input.objectPath,

    isPublic:
      normalizeContentIsPublic(
        input.isPublic,
      ),

    size:
      Number.isFinite(input.size) &&
      input.size >= 0
        ? input.size
        : 0,

    createdAt: input.createdAt,
    createdBy: input.createdBy,

    updatedAt: input.updatedAt,
    updatedBy: input.updatedBy,
  };

  const errors =
    validateContentFile(normalized);

  if (errors.length > 0) {
    throw new Error(
      `Invalid ContentFile: ${errors.join(
        ", ",
      )}`,
    );
  }

  return normalized;
}

export function validateContentFiles(
  contents: ContentFile[],
): string[] {
  const errors: string[] = [];

  const ids = new Set<string>();

  const objectPaths =
    new Set<string>();

  contents.forEach(
    (
      content,
      index,
    ) => {
      const contentErrors =
        validateContentFile(content);

      for (
        const error of contentErrors
      ) {
        errors.push(
          `contentFiles[${index}].${error}`,
        );
      }

      if (content.id) {
        if (ids.has(content.id)) {
          errors.push(
            `contentFiles[${index}].id duplicated`,
          );
        }

        ids.add(content.id);
      }

      if (content.objectPath) {
        if (
          objectPaths.has(
            content.objectPath,
          )
        ) {
          errors.push(
            `contentFiles[${index}].objectPath duplicated`,
          );
        }

        objectPaths.add(
          content.objectPath,
        );
      }
    },
  );

  return errors;
}

/* =========================================================
 * TokenIcon
 * =======================================================*/

export interface TokenIcon {
  id: string;
  url: string;
  objectPath: string;
  fileName: string;
  contentType: string;
  size: number;
}

/* =========================================================
 * Token icon policy
 * =======================================================*/

export const TOKEN_ICON_MAX_FILE_SIZE =
  10 * 1024 * 1024;

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

/**
 * ファイル名の拡張子がTokenIconの
 * 許可対象か判定する。
 *
 * TOKEN_ICON_ALLOWED_EXTENSIONSは
 * 固定の非空配列なので、
 * length === 0の判定は行わない。
 */
export function isTokenIconExtensionAllowed(
  fileName: string,
): boolean {
  const normalized =
    fileName.toLowerCase();

  return TOKEN_ICON_ALLOWED_EXTENSIONS.some(
    (extension) => {
      return normalized.endsWith(
        extension,
      );
    },
  );
}

export function isTokenIconContentTypeAllowed(
  contentType: string,
): boolean {
  const normalized =
    contentType.toLowerCase();

  return TOKEN_ICON_ALLOWED_CONTENT_TYPES.some(
    (allowed) => {
      return allowed === normalized;
    },
  );
}

export function validateTokenIcon(
  icon: TokenIcon,
): boolean {
  if (!icon.id) {
    return false;
  }

  if (
    !icon.url ||
    !isValidHttpUrl(icon.url)
  ) {
    return false;
  }

  if (!icon.objectPath) {
    return false;
  }

  if (!icon.fileName) {
    return false;
  }

  if (
    !icon.contentType ||
    !isTokenIconContentTypeAllowed(
      icon.contentType,
    )
  ) {
    return false;
  }

  if (
    !Number.isFinite(icon.size) ||
    icon.size < 0
  ) {
    return false;
  }

  if (
    icon.size >
    TOKEN_ICON_MAX_FILE_SIZE
  ) {
    return false;
  }

  return true;
}

export function validateTokenIconFile(
  file: File,
): boolean {
  if (!file) {
    return false;
  }

  if (
    !file.name ||
    !isTokenIconExtensionAllowed(
      file.name,
    )
  ) {
    return false;
  }

  if (
    !file.type ||
    !isTokenIconContentTypeAllowed(
      file.type,
    )
  ) {
    return false;
  }

  if (
    !Number.isFinite(file.size) ||
    file.size < 0
  ) {
    return false;
  }

  if (
    file.size >
    TOKEN_ICON_MAX_FILE_SIZE
  ) {
    return false;
  }

  return true;
}

export function createTokenIcon(
  input: TokenIcon,
): TokenIcon {
  const normalized: TokenIcon = {
    id: input.id,
    url: input.url,
    objectPath: input.objectPath,
    fileName: input.fileName,
    contentType:
      input.contentType,

    size:
      Number.isFinite(input.size) &&
      input.size >= 0
        ? input.size
        : 0,
  };

  if (
    !validateTokenIcon(
      normalized,
    )
  ) {
    throw new Error(
      "Invalid TokenIcon",
    );
  }

  return normalized;
}

/* =========================================================
 * TokenBlueprint
 * =======================================================*/

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

export function validateTokenBlueprint(
  input: TokenBlueprint,
): string[] {
  const errors: string[] = [];

  if (!input.id) {
    errors.push("id is required");
  }

  if (!input.name) {
    errors.push("name is required");
  }

  if (!input.symbol) {
    errors.push(
      "symbol is required",
    );
  }

  if (!input.brandId) {
    errors.push(
      "brandId is required",
    );
  }

  if (!input.companyId) {
    errors.push(
      "companyId is required",
    );
  }

  if (!input.assigneeId) {
    errors.push(
      "assigneeId is required",
    );
  }

  const hasAnyIconField =
    Boolean(input.iconUrl) ||
    Boolean(
      input.iconObjectPath,
    ) ||
    Boolean(
      input.iconFileName,
    ) ||
    Boolean(
      input.iconContentType,
    ) ||
    input.iconSize != null;

  if (hasAnyIconField) {
    if (!input.iconUrl) {
      errors.push(
        "iconUrl is required when icon is set",
      );
    }

    if (
      input.iconUrl &&
      !isValidHttpUrl(
        input.iconUrl,
      )
    ) {
      errors.push(
        "iconUrl must be a valid http(s) URL",
      );
    }

    if (!input.iconObjectPath) {
      errors.push(
        "iconObjectPath is required when icon is set",
      );
    }

    if (!input.iconFileName) {
      errors.push(
        "iconFileName is required when icon is set",
      );
    }

    if (
      input.iconSize != null &&
      (
        !Number.isFinite(
          input.iconSize,
        ) ||
        input.iconSize < 0
      )
    ) {
      errors.push(
        "iconSize must be 0 or greater",
      );
    }
  }

  errors.push(
    ...validateContentFiles(
      input.contentFiles ?? [],
    ),
  );

  return errors;
}

/* =========================================================
 * Firebase Storage delete operation
 * =======================================================*/

export interface FirebaseStorageDeleteOp {
  objectPath: string;
}

export function toContentFileFirebaseStorageDeleteOp(
  content: ContentFile,
): FirebaseStorageDeleteOp {
  return {
    objectPath:
      content.objectPath,
  };
}

export function toTokenIconFirebaseStorageDeleteOp(
  icon: TokenIcon,
): FirebaseStorageDeleteOp {
  return {
    objectPath:
      icon.objectPath,
  };
}

/* =========================================================
 * Internal helpers
 * =======================================================*/

function isValidHttpUrl(
  raw: string,
): boolean {
  try {
    const url = new URL(raw);

    if (
      !url.protocol ||
      !url.hostname
    ) {
      return false;
    }

    return (
      url.protocol === "http:" ||
      url.protocol === "https:"
    );
  } catch {
    return false;
  }
}