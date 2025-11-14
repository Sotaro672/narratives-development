// frontend/shell/src/shared/types/listImage.ts

/**
 * ListImage
 * backend/internal/domain/listImage/entity.go の ListImage に対応。
 *
 * - 日付は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）を想定
 * - updatedAt / updatedBy / deletedAt / deletedBy は省略可
 * - displayOrder は 0 以上の整数
 */
export interface ListImage {
  id: string;
  listId: string;
  url: string;
  fileName: string;
  size: number;
  displayOrder: number;

  createdAt: string;
  createdBy: string;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * 画像ファイルの検証結果（UI 向け）
 * backend/internal/domain/listImage/entity.go の ImageFileValidation に対応。
 */
export interface ImageFileValidation {
  isValid: boolean;
  errorMessage?: string;
}

/**
 * UI 向けエラーメッセージ（backend と整合）
 */
export const ERR_MSG_INVALID_FILE_TYPE = "無効なファイル形式です";
export const ERR_MSG_FILE_TOO_LARGE = "ファイルサイズが大きすぎます";
export const ERR_MSG_UPLOAD_FAILED = "画像のアップロードに失敗しました";

/**
 * Policy
 * backend/internal/domain/listImage/entity.go の定数と概ね整合させる。
 */

// data URL 検証用の上限（service 層などで使用）
export const DEFAULT_MAX_IMAGE_SIZE_BYTES = 5 * 1024 * 1024; // 5MB

// アップロード許可 MIME タイプ
export const SUPPORTED_IMAGE_MIMES = new Set<string>([
  "image/jpeg",
  "image/jpg",
  "image/png",
  "image/webp",
]);

// ListImage 実体用の許可拡張子（空なら制限なし）
export const ALLOWED_LIST_IMAGE_EXTENSIONS = new Set<string>([
  ".png",
  ".jpg",
  ".jpeg",
  ".webp",
  ".gif",
]);

// ListImage 実体の最大ファイルサイズ（0 なら上限なし）
export const MAX_LIST_IMAGE_FILE_SIZE = 20 * 1024 * 1024; // 20MB

/**
 * 必須文字列チェック（必要に応じて呼び出し側で使用）
 */
export function requireNonEmpty(name: string, v: string): Error | null {
  if (!v || v.trim() === "") {
    return new Error(`${name} is required`);
  }
  return null;
}

/**
 * data URL 形式 (data:<mime>;base64,...) の簡易検証
 * backend の ValidateDataURL と整合する範囲でフロント向け実装。
 * - 戻り値: mime と base64 デコード済み payload
 * - バリデーションエラー時は例外を投げる
 */
export function validateDataUrl(
  data: string,
  maxBytes: number = DEFAULT_MAX_IMAGE_SIZE_BYTES,
  supported: Set<string> = SUPPORTED_IMAGE_MIMES
): { mime: string; payload: Uint8Array } {
  if (!data.startsWith("data:")) {
    throw new Error("invalid data URL: missing 'data:' prefix");
  }

  const [meta, raw] = data.split(",", 2);
  if (!raw) {
    throw new Error("invalid data URL: missing payload");
  }
  if (!meta.includes(";base64")) {
    throw new Error("invalid data URL: not base64 encoded");
  }

  const mime = meta.substring("data:".length, meta.indexOf(";base64"));
  if (!mime) {
    throw new Error("invalid data URL: missing mime type");
  }
  if (!supported.has(mime)) {
    throw new Error(`unsupported content type: ${mime}`);
  }

  let binary: string;
  try {
    // ブラウザ環境想定
    binary = atob(raw);
  } catch {
    throw new Error("invalid base64 payload");
  }

  const len = binary.length;
  if (len === 0) {
    throw new Error("empty image payload");
  }
  if (len > maxBytes) {
    throw new Error(`file too large: ${len} bytes (max ${maxBytes})`);
  }

  const bytes = new Uint8Array(len);
  for (let i = 0; i < len; i++) {
    bytes[i] = binary.charCodeAt(i);
  }

  return { mime, payload: bytes };
}

/**
 * URL の簡易バリデーション
 * backend の validateURL と整合する範囲。
 */
export function isValidListImageUrl(u: string): boolean {
  if (!u || !u.trim()) return false;
  try {
    const parsed = new URL(u);
    return !!parsed.protocol && !!parsed.host;
  } catch {
    return false;
  }
}

/**
 * 拡張子チェック（ListImage 用）
 */
export function isAllowedListImageExtension(fileName: string): boolean {
  if (!fileName) return false;
  if (ALLOWED_LIST_IMAGE_EXTENSIONS.size === 0) return true;
  const idx = fileName.lastIndexOf(".");
  if (idx < 0) return false;
  const ext = fileName.slice(idx).toLowerCase();
  return ALLOWED_LIST_IMAGE_EXTENSIONS.has(ext);
}

/**
 * ListImage の簡易バリデーション
 * backend/internal/domain/listImage/entity.go の validate() と整合する範囲。
 */
export function validateListImage(li: ListImage): boolean {
  // id / listId
  if (!li.id || !li.id.trim()) return false;
  if (!li.listId || !li.listId.trim()) return false;

  // url
  if (!isValidListImageUrl(li.url)) return false;

  // fileName
  if (!li.fileName || !isAllowedListImageExtension(li.fileName)) return false;

  // size
  if (typeof li.size !== "number" || li.size < 0) return false;
  if (MAX_LIST_IMAGE_FILE_SIZE > 0 && li.size > MAX_LIST_IMAGE_FILE_SIZE) {
    return false;
  }

  // displayOrder
  if (!Number.isInteger(li.displayOrder) || li.displayOrder < 0) return false;

  // createdAt / createdBy
  if (!li.createdAt || Number.isNaN(Date.parse(li.createdAt))) return false;
  if (!li.createdBy || !li.createdBy.trim()) return false;

  // updatedAt
  if (
    li.updatedAt != null &&
    li.updatedAt !== "" &&
    Number.isNaN(Date.parse(li.updatedAt))
  ) {
    return false;
  }

  // deletedAt
  if (
    li.deletedAt != null &&
    li.deletedAt !== "" &&
    Number.isNaN(Date.parse(li.deletedAt))
  ) {
    return false;
  }

  return true;
}
