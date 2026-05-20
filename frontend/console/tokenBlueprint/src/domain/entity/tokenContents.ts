// frontend/console/tokenBlueprint/src/domain/entity/tokenContents.ts

/**
 * TokenContents domain entity
 *
 * Firebase Storage 移行後の正仕様:
 * - url は Firebase Storage downloadURL
 * - objectPath は Firebase Storage object path
 * - GCS bucket / public GCS URL / signed URL / gs:// URL は扱わない
 */

/**
 * ContentType
 * backend/internal/domain/tokenContents/entity.go の ContentType に対応。
 *
 * - "image" | "video" | "pdf" | "document"
 */
export type ContentType = "image" | "video" | "pdf" | "document";

/** ContentType の妥当性チェック */
export function isValidContentType(t: string): t is ContentType {
  return t === "image" || t === "video" || t === "pdf" || t === "document";
}

/** 利用可能な ContentType 一覧（UI用） */
export const ALL_CONTENT_TYPES: ContentType[] = [
  "image",
  "video",
  "pdf",
  "document",
];

/**
 * FirebaseStorageTokenContent
 *
 * Firebase Storage に保存された token content の表示・削除に必要な最小情報。
 */
export interface FirebaseStorageTokenContent {
  id: string;
  name: string;
  type: ContentType;
  contentType: string;
  size: number;
  objectPath: string;
  url: string;
}

/**
 * Firebase Storage delete operation.
 *
 * Firebase Storage の削除対象は objectPath のみで特定する。
 */
export interface FirebaseStorageDeleteOp {
  objectPath: string;
}

/* =========================================================
 * バリデーション / ヘルパ
 * =======================================================*/

/** アップロードポリシー（MaxFileSize: 0 なら上限なし） */
export const MAX_TOKEN_CONTENT_FILE_SIZE = 50 * 1024 * 1024; // 50MB

/** FirebaseStorageTokenContent の簡易バリデーション */
export function validateFirebaseStorageTokenContent(
  c: FirebaseStorageTokenContent,
): string[] {
  const errors: string[] = [];

  if (!c.id.trim()) errors.push("id is required");
  if (!c.name.trim()) errors.push("name is required");

  if (!isValidContentType(c.type)) {
    errors.push("type must be one of 'image' | 'video' | 'pdf' | 'document'");
  }

  if (!c.contentType.trim()) {
    errors.push("contentType is required");
  }

  if (c.size < 0) {
    errors.push("size must be >= 0");
  } else if (
    MAX_TOKEN_CONTENT_FILE_SIZE > 0 &&
    c.size > MAX_TOKEN_CONTENT_FILE_SIZE
  ) {
    errors.push(`size must be <= ${MAX_TOKEN_CONTENT_FILE_SIZE} bytes`);
  }

  if (!c.objectPath.trim()) {
    errors.push("objectPath is required");
  }

  if (!c.url.trim()) {
    errors.push("url is required");
  } else {
    try {
      const parsed = new URL(c.url);

      if (!/^https?:$/i.test(parsed.protocol)) {
        errors.push("url must be http(s)");
      }
    } catch {
      errors.push("url must be a valid URL");
    }
  }

  return errors;
}

/**
 * 正規化付きファクトリ
 * - 文字列を trim
 * - URL / objectPath / size / type 等を validate
 * - エラー時は例外
 */
export function createFirebaseStorageTokenContent(
  input: FirebaseStorageTokenContent,
): FirebaseStorageTokenContent {
  const normalized: FirebaseStorageTokenContent = {
    id: input.id.trim(),
    name: input.name.trim(),
    type: input.type,
    contentType: input.contentType.trim(),
    size: input.size,
    objectPath: input.objectPath.trim(),
    url: input.url.trim(),
  };

  const errors = validateFirebaseStorageTokenContent(normalized);

  if (errors.length > 0) {
    throw new Error(
      `Invalid FirebaseStorageTokenContent: ${errors.join(", ")}`,
    );
  }

  return normalized;
}

/* =========================================================
 * Firebase Storage 削除オペレーション ヘルパ
 * =======================================================*/

/**
 * ToFirebaseStorageDeleteOp
 *
 * Firebase Storage では bucket を UI/domain で扱わず、
 * objectPath のみを削除対象として渡す。
 */
export function toFirebaseStorageDeleteOp(
  content: FirebaseStorageTokenContent,
): FirebaseStorageDeleteOp {
  return {
    objectPath: content.objectPath,
  };
}