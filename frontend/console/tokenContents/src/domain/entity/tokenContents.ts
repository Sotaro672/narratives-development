// frontend/tokenContents/src/domain/entity/tokenContents.ts

/**
 * Default GCS bucket for TokenContents files.
 * backend/internal/domain/tokenContents/entity.go の DefaultBucket に対応。
 */
export const DEFAULT_TOKEN_CONTENTS_BUCKET =
  "narratives_development_token_contents";

/**
 * GCSDeleteOp
 * backend/internal/domain/tokenContents/entity.go の GCSDeleteOp に対応。
 */
export interface GCSDeleteOp {
  bucket: string;
  objectPath: string;
}

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
 * GCSTokenContent
 * backend/internal/domain/tokenContents/entity.go の GCSTokenContent に対応。
 *
 * - createdAt などの監査情報は backend 側から削除されているためここでも持たない
 * - URL は公開アクセス可能な HTTPS URL を想定
 */
export interface GCSTokenContent {
  id: string;
  name: string;
  type: ContentType;
  url: string;
  /** ファイルサイズ（bytes, 0以上） */
  size: number;
}

/* =========================================================
 * バリデーション / ヘルパ
 * =======================================================*/

/** アップロードポリシー（MaxFileSize: 0 なら上限なし） */
export const MAX_TOKEN_CONTENT_FILE_SIZE = 50 * 1024 * 1024; // 50MB

/** GCSTokenContent の簡易バリデーション（Go側 validate() と対応） */
export function validateGCSTokenContent(
  c: GCSTokenContent,
): string[] {
  const errors: string[] = [];

  if (!c.id?.trim()) errors.push("id is required");
  if (!c.name?.trim()) errors.push("name is required");

  if (!isValidContentType(c.type)) {
    errors.push(
      "type must be one of 'image' | 'video' | 'pdf' | 'document'",
    );
  }

  if (!c.url?.trim()) {
    errors.push("url is required");
  } else {
    try {
      const u = new URL(c.url);
      if (!/^https?:$/i.test(u.protocol)) {
        errors.push("url must be http(s)");
      }
    } catch {
      errors.push("url must be a valid URL");
    }
  }

  if (c.size == null || Number.isNaN(c.size)) {
    errors.push("size is required");
  } else if (c.size < 0) {
    errors.push("size must be >= 0");
  } else if (
    MAX_TOKEN_CONTENT_FILE_SIZE > 0 &&
    c.size > MAX_TOKEN_CONTENT_FILE_SIZE
  ) {
    errors.push(
      `size must be <= ${MAX_TOKEN_CONTENT_FILE_SIZE} bytes`,
    );
  }

  return errors;
}

/**
 * 正規化付きファクトリ
 * - 文字列を trim
 * - URL/size/type 等を validate（エラー時は例外を投げる想定の軽量版）
 */
export function createGCSTokenContent(
  input: GCSTokenContent,
): GCSTokenContent {
  const normalized: GCSTokenContent = {
    ...input,
    id: input.id.trim(),
    name: input.name.trim(),
    type: input.type,
    url: input.url.trim(),
    size: input.size,
  };

  const errors = validateGCSTokenContent(normalized);
  if (errors.length > 0) {
    throw new Error(
      `Invalid GCSTokenContent: ${errors.join(", ")}`,
    );
  }
  return normalized;
}

/* =========================================================
 * GCS URL / 削除オペレーション ヘルパ
 * =======================================================*/

/**
 * PublicURL
 * backend/internal/domain/tokenContents/entity.go の PublicURL に対応。
 * https://storage.googleapis.com/{bucket}/{objectPath}
 */
export function publicURL(
  bucket: string,
  objectPath: string,
): string {
  const b = (bucket || "").trim() || DEFAULT_TOKEN_CONTENTS_BUCKET;
  const obj = (objectPath || "").trim().replace(/^\/+/, "");
  return `https://storage.googleapis.com/${b}/${obj}`;
}

/**
 * ParseGCSURL
 * backend/internal/domain/tokenContents/entity.go の ParseGCSURL に対応。
 *
 * 対応形式:
 * - https://storage.googleapis.com/{bucket}/{objectPath}
 * - https://storage.cloud.google.com/{bucket}/{objectPath}
 *
 * パース成功時: { bucket, objectPath, ok: true }
 * 失敗時: { bucket: "", objectPath: "", ok: false }
 */
export function parseGCSURL(u: string): {
  bucket: string;
  objectPath: string;
  ok: boolean;
} {
  let parsed: URL;
  try {
    parsed = new URL((u || "").trim());
  } catch {
    return { bucket: "", objectPath: "", ok: false };
  }

  const host = parsed.host.toLowerCase();
  if (
    host !== "storage.googleapis.com" &&
    host !== "storage.cloud.google.com"
  ) {
    return { bucket: "", objectPath: "", ok: false };
  }

  const path = parsed.pathname.replace(/^\/+/, "");
  if (!path) return { bucket: "", objectPath: "", ok: false };

  const [bucket, ...rest] = path.split("/");
  if (!bucket || rest.length === 0) {
    return { bucket: "", objectPath: "", ok: false };
  }

  const objectPath = decodeURIComponent(rest.join("/"));
  return { bucket, objectPath, ok: true };
}

/**
 * ToGCSDeleteOp
 * backend/internal/domain/tokenContents/entity.go の ToGCSDeleteOp に対応。
 *
 * 優先順:
 * 1. URL から GCS の bucket/object を解決できればそれを使用
 * 2. それ以外は DefaultBucket + name を objectPath として扱う
 */
export function toGCSDeleteOp(
  content: GCSTokenContent,
): GCSDeleteOp {
  const { bucket, objectPath, ok } = parseGCSURL(content.url);
  if (ok) {
    return { bucket, objectPath };
  }
  return {
    bucket: DEFAULT_TOKEN_CONTENTS_BUCKET,
    objectPath: (content.name || "").trim().replace(/^\/+/, ""),
  };
}
