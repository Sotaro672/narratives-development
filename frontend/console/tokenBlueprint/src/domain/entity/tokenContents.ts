// frontend/console/tokenBlueprint/src/domain/entity/tokenContents.ts

/**
 * Default GCS bucket for TokenContents files.
 * backend/internal/domain/tokenContents/entity.go の DefaultBucket を正とする。
 *
 * 注意:
 * - GCS の bucket 名は "_" ではなく "-" を推奨（GCS 命名規則に沿う）
 * - 現行の運用: narratives-development-token-contents
 */
export const DEFAULT_TOKEN_CONTENTS_BUCKET =
  "narratives-development-token-contents";

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
export const ALL_CONTENT_TYPES: ContentType[] = ["image", "video", "pdf", "document"];

/**
 * GCSTokenContent
 * backend/internal/domain/tokenContents/entity.go の GCSTokenContent に対応。
 *
 * - URL は公開アクセス可能な HTTPS URL を想定（署名付き URL を扱う場合は別途要件整理）
 * - id は objectPath として運用される想定（GCS Object 名）
 */
export interface GCSTokenContent {
  /** 通常: objectPath（例: "1234567890_file.pdf"） */
  id: string;
  name: string;
  type: ContentType;
  /** 例: https://storage.googleapis.com/{bucket}/{objectPath} */
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
export function validateGCSTokenContent(c: GCSTokenContent): string[] {
  const errors: string[] = [];

  if (!c.id?.trim()) errors.push("id is required");
  if (!c.name?.trim()) errors.push("name is required");

  if (!isValidContentType(c.type)) {
    errors.push("type must be one of 'image' | 'video' | 'pdf' | 'document'");
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
  } else if (MAX_TOKEN_CONTENT_FILE_SIZE > 0 && c.size > MAX_TOKEN_CONTENT_FILE_SIZE) {
    errors.push(`size must be <= ${MAX_TOKEN_CONTENT_FILE_SIZE} bytes`);
  }

  return errors;
}

/**
 * 正規化付きファクトリ
 * - 文字列を trim
 * - URL/size/type 等を validate（エラー時は例外）
 */
export function createGCSTokenContent(input: GCSTokenContent): GCSTokenContent {
  const normalized: GCSTokenContent = {
    ...input,
    id: String(input.id ?? "").trim(),
    name: String(input.name ?? "").trim(),
    type: input.type,
    url: String(input.url ?? "").trim(),
    size: Number(input.size ?? 0),
  };

  const errors = validateGCSTokenContent(normalized);
  if (errors.length > 0) {
    throw new Error(`Invalid GCSTokenContent: ${errors.join(", ")}`);
  }
  return normalized;
}

/* =========================================================
 * GCS URL / 削除オペレーション ヘルパ
 * =======================================================*/

/**
 * PublicURL
 * backend/internal/domain/tokenContents/entity.go の PublicURL を正とする。
 * https://storage.googleapis.com/{bucket}/{objectPath}
 */
export function publicURL(bucket: string, objectPath: string): string {
  const b = (bucket || "").trim() || DEFAULT_TOKEN_CONTENTS_BUCKET;
  const obj = (objectPath || "").trim().replace(/^\/+/, "");
  return `https://storage.googleapis.com/${b}/${encodeURIComponentPath(obj)}`;
}

/**
 * ParseGCSURL
 * backend 側の ParseGCSURL と同等の入力許容を目指す。
 *
 * 対応形式（代表）:
 * - gs://{bucket}/{objectPath}
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
  const raw = (u || "").trim();
  if (!raw) return { bucket: "", objectPath: "", ok: false };

  // gs://bucket/object
  if (raw.toLowerCase().startsWith("gs://")) {
    const rest = raw.slice("gs://".length);
    const cleaned = rest.replace(/^\/+/, "");
    const [bucket, ...parts] = cleaned.split("/");
    if (!bucket || parts.length === 0) return { bucket: "", objectPath: "", ok: false };
    return { bucket, objectPath: decodeURIComponent(parts.join("/")), ok: true };
  }

  // https://...
  let parsed: URL;
  try {
    parsed = new URL(raw);
  } catch {
    return { bucket: "", objectPath: "", ok: false };
  }

  const host = parsed.host.toLowerCase();
  if (host !== "storage.googleapis.com" && host !== "storage.cloud.google.com") {
    return { bucket: "", objectPath: "", ok: false };
  }

  const path = parsed.pathname.replace(/^\/+/, "");
  if (!path) return { bucket: "", objectPath: "", ok: false };

  const [bucket, ...rest] = path.split("/");
  if (!bucket || rest.length === 0) return { bucket: "", objectPath: "", ok: false };

  const objectPath = decodeURIComponent(rest.join("/"));
  if (!objectPath) return { bucket: "", objectPath: "", ok: false };

  return { bucket, objectPath, ok: true };
}

/**
 * ToGCSDeleteOp
 * backend/internal/domain/tokenContents/entity.go の ToGCSDeleteOp を正とする。
 *
 * 優先順（UI 側の事故を減らす）:
 * 1. url が GCS URL としてパースできる → それを使用
 * 2. それ以外は DefaultBucket + id（id は objectPath として扱う）
 * 3. id が空なら DefaultBucket + name（最終フォールバック）
 */
export function toGCSDeleteOp(content: GCSTokenContent): GCSDeleteOp {
  const { bucket, objectPath, ok } = parseGCSURL(content.url);
  if (ok) {
    return { bucket, objectPath };
  }

  const idPath = String(content.id ?? "").trim().replace(/^\/+/, "");
  if (idPath) {
    return { bucket: DEFAULT_TOKEN_CONTENTS_BUCKET, objectPath: idPath };
  }

  const namePath = String(content.name ?? "").trim().replace(/^\/+/, "");
  return { bucket: DEFAULT_TOKEN_CONTENTS_BUCKET, objectPath: namePath };
}

/* =========================================================
 * Internal
 * =======================================================*/

/**
 * encodeURIComponent は "/" もエンコードしてしまうため、パスセグメントは維持してエンコードする
 */
function encodeURIComponentPath(path: string): string {
  return path
    .split("/")
    .map((seg) => encodeURIComponent(seg))
    .join("/");
}
