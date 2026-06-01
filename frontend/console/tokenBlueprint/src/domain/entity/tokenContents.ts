// frontend/console/tokenBlueprint/src/domain/entity/tokenContents.ts

/**
 * TokenContents domain entity
 *
 * Firebase Storage 移行後の正仕様:
 * - url は Firebase Storage downloadURL
 * - backend/domain では objectPath / name / size を保持しない
 * - backend は url だけで content file を制御する
 */

/**
 * ContentType
 * backend/internal/domain/tokenBlueprint/entity.go の ContentFileType に対応。
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
 * ContentVisibility
 * backend/internal/domain/tokenBlueprint/entity.go の ContentVisibility に対応。
 */
export type ContentVisibility = "private" | "public";

/** ContentVisibility の妥当性チェック */
export function isValidContentVisibility(v: string): v is ContentVisibility {
  return v === "private" || v === "public";
}

/**
 * FirebaseStorageTokenContent
 *
 * Firebase Storage に保存された token content の表示に必要な最小情報。
 *
 * backend ContentFile struct と一致:
 * - id
 * - type
 * - contentType
 * - url
 * - visibility
 * - createdAt
 * - createdBy
 * - updatedAt
 * - updatedBy
 */
export interface FirebaseStorageTokenContent {
  id: string;
  type: ContentType;
  contentType: string;
  url: string;
  visibility: ContentVisibility;

  createdAt?: string;
  createdBy?: string;
  updatedAt?: string;
  updatedBy?: string;
}

/**
 * Firebase Storage delete operation.
 *
 * objectPath は廃止済み。
 * 削除・制御は backend 側で url を基準に行う。
 */
export interface FirebaseStorageDeleteOp {
  url: string;
}

/* =========================================================
 * バリデーション / ヘルパ
 * =======================================================*/

/** FirebaseStorageTokenContent の簡易バリデーション */
export function validateFirebaseStorageTokenContent(
  c: FirebaseStorageTokenContent,
): string[] {
  const errors: string[] = [];

  if (!c.id.trim()) {
    errors.push("id is required");
  }

  if (!isValidContentType(c.type)) {
    errors.push("type must be one of 'image' | 'video' | 'pdf' | 'document'");
  }

  if (c.contentType && !c.contentType.trim()) {
    errors.push("contentType must not be blank when provided");
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

  if (!isValidContentVisibility(c.visibility)) {
    errors.push("visibility must be one of 'private' | 'public'");
  }

  return errors;
}

/**
 * 正規化付きファクトリ
 * - 文字列を trim
 * - URL / type / visibility 等を validate
 * - エラー時は例外
 */
export function createFirebaseStorageTokenContent(
  input: FirebaseStorageTokenContent,
): FirebaseStorageTokenContent {
  const normalized: FirebaseStorageTokenContent = {
    id: input.id.trim(),
    type: input.type,
    contentType: input.contentType.trim(),
    url: input.url.trim(),
    visibility: input.visibility,

    createdAt: input.createdAt?.trim(),
    createdBy: input.createdBy?.trim(),
    updatedAt: input.updatedAt?.trim(),
    updatedBy: input.updatedBy?.trim(),
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
 * objectPath は廃止済み。
 * backend 側で url を基準に削除・制御する。
 */
export function toFirebaseStorageDeleteOp(
  content: FirebaseStorageTokenContent,
): FirebaseStorageDeleteOp {
  return {
    url: content.url,
  };
}