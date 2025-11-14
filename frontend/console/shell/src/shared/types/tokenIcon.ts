// frontend/shell/src/shared/types/tokenIcon.ts
// Generated from frontend/tokenBlueprint/src/domain/entity/tokenIcon.tsx
// backend/internal/domain/tokenIcon/entity.go に対応する共通型定義

/**
 * Default GCS bucket for TokenIcon files.
 * backend の DefaultBucket と同期。
 */
export const TOKEN_ICON_DEFAULT_BUCKET =
  "narratives_development_token_icon";

/**
 * GCSDeleteOp
 * GCS 上オブジェクト削除時の指定用。
 */
export interface GCSDeleteOp {
  bucket: string;
  objectPath: string;
}

/**
 * TokenIcon
 * backend/internal/domain/tokenIcon/entity.go の TokenIcon に対応。
 *
 * - url: 公開 / 参照用 URL
 * - fileName: 元ファイル名
 * - size: バイト数
 */
export interface TokenIcon {
  id: string;
  url: string;
  fileName: string;
  size: number;
}

/**
 * Policy (backend と同期)
 */
export const TOKEN_ICON_MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB

export const TOKEN_ICON_ALLOWED_EXTENSIONS = [
  ".png",
  ".jpg",
  ".jpeg",
  ".webp",
  ".gif",
] as const;

/**
 * TokenIcon の簡易バリデーション
 * backend の validate() と整合する範囲でチェック。
 */
export function validateTokenIcon(icon: TokenIcon): boolean {
  // id
  if (!icon.id?.trim()) return false;

  // url
  if (!icon.url?.trim() || !isValidUrl(icon.url)) return false;

  // fileName
  if (!icon.fileName?.trim()) return false;

  // size
  if (!Number.isFinite(icon.size) || icon.size < 0) return false;
  if (TOKEN_ICON_MAX_FILE_SIZE > 0 && icon.size > TOKEN_ICON_MAX_FILE_SIZE) {
    return false;
  }

  return true;
}

/**
 * ファイル名の拡張子が許可されているか判定
 */
export function isTokenIconExtensionAllowed(fileName: string): boolean {
  if (!TOKEN_ICON_ALLOWED_EXTENSIONS.length) return true;
  const lower = fileName.toLowerCase();
  return TOKEN_ICON_ALLOWED_EXTENSIONS.some((ext) => lower.endsWith(ext));
}

/**
 * PublicURL
 * GCS オブジェクトの公開 URL を生成
 */
export function buildTokenIconPublicURL(
  bucket: string,
  objectPath: string,
): string {
  const b = (bucket || TOKEN_ICON_DEFAULT_BUCKET).trim();
  const obj = objectPath.replace(/^\/+/, "").trim();
  return `https://storage.googleapis.com/${b}/${obj}`;
}

/**
 * ParseGCSURL
 * GCS URL から bucket と objectPath を抽出。
 */
export function parseTokenIconGCSURL(
  rawUrl: string,
): { bucket: string; objectPath: string } | null {
  try {
    const u = new URL(rawUrl.trim());
    const host = u.hostname.toLowerCase();
    if (
      host !== "storage.googleapis.com" &&
      host !== "storage.cloud.google.com"
    ) {
      return null;
    }

    const path = u.pathname.replace(/^\/+/, "");
    if (!path) return null;

    const [bucket, ...rest] = path.split("/");
    if (!bucket || rest.length === 0) return null;

    const objectPath = rest.join("/");
    return { bucket, objectPath };
  } catch {
    return null;
  }
}

/**
 * ToGCSDeleteOp 相当の関数。
 * - URL から GCS パスを解析できればそれを利用
 * - できない場合は DefaultBucket + "token_icons/{fileName}" を利用
 */
export function toTokenIconGCSDeleteOp(icon: TokenIcon): GCSDeleteOp {
  const parsed = parseTokenIconGCSURL(icon.url);
  if (parsed) {
    return { bucket: parsed.bucket, objectPath: parsed.objectPath };
  }
  return {
    bucket: TOKEN_ICON_DEFAULT_BUCKET,
    objectPath: `token_icons/${icon.fileName.trim()}`,
  };
}

// ==============================
// internal helper
// ==============================

function isValidUrl(raw: string): boolean {
  try {
    const u = new URL(raw.trim());
    if (!u.protocol || !u.hostname) return false;
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}
