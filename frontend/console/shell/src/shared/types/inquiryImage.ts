// frontend/shell/src/shared/types/inquiryImage.ts

/**
 * inquiryImage / ImageFile shared types
 * backend/internal/domain/inquiryImage/entity.go
 * および frontend/inquiry/src/domain/entity/inquiryImage.ts に対応。
 *
 * 他マイクロフロントエンドから参照される共通型として利用します。
 */

/**
 * Default GCS bucket for inquiry images
 * backend の DefaultBucket と揃える。
 */
export const INQUIRY_IMAGE_DEFAULT_BUCKET =
  "narratives_development_inquiry_image";

/**
 * GCSDeleteOp
 * GCS 上のオブジェクト削除指示用。
 */
export interface GCSDeleteOp {
  bucket: string;
  objectPath: string;
}

/**
 * ImageFile
 * 問い合わせに紐づく画像ファイル。
 *
 * backend/internal/domain/inquiryImage/entity.go の ImageFile に対応。
 * - 日付は ISO8601 文字列
 * - width / height は省略可
 * - updated*, deleted* は任意
 */
export interface ImageFile {
  inquiryId: string;
  fileName: string;
  fileUrl: string;
  fileSize: number;
  mimeType: string;
  width?: number;
  height?: number;
  createdAt: string;
  createdBy: string;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * InquiryImage
 * 問い合わせIDと、その問い合わせに紐づく画像一覧の集約。
 */
export interface InquiryImage {
  id: string; // = inquiryId
  images: ImageFile[];
}

/**
 * InquiryImagePatch
 * 部分更新用 DTO。
 * backend の InquiryImagePatch (UpdatedBy/DeletedAt/DeletedBy) に対応。
 */
export interface InquiryImagePatch {
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * Policy（backend と同期）
 */
export const INQUIRY_IMAGE_MAX_IMAGES = 10;
export const INQUIRY_IMAGE_MIN_FILE_SIZE = 1; // >= 1 byte
export const INQUIRY_IMAGE_MAX_FILE_SIZE = 20 * 1024 * 1024; // 20MB
export const INQUIRY_IMAGE_MAX_FILE_NAME_LENGTH = 255;

export const INQUIRY_IMAGE_ALLOWED_MIME_TYPES: string[] = [
  "image/jpeg",
  "image/png",
  "image/webp",
  "image/gif",
];

// 空配列の場合は全ホスト許可（backend と同じ意味付け）
export const INQUIRY_IMAGE_ALLOWED_URL_HOSTS: string[] = [];

// MIME の形式チェック用（厳密判定は backend 側に委譲）
const MIME_REGEX = /^[a-zA-Z0-9.+-]+\/[a-zA-Z0-9.+-]+$/;

/**
 * ImageFile の簡易バリデーション
 * backend の validateImageFile に合わせられる範囲でチェック。
 */
export function validateImageFile(image: ImageFile): boolean {
  // inquiryId
  if (!image.inquiryId?.trim()) return false;

  // fileName
  if (!image.fileName?.trim()) return false;
  if (
    INQUIRY_IMAGE_MAX_FILE_NAME_LENGTH > 0 &&
    [...image.fileName].length > INQUIRY_IMAGE_MAX_FILE_NAME_LENGTH
  ) {
    return false;
  }

  // fileUrl
  if (!isValidUrl(image.fileUrl)) return false;

  // fileSize
  if (
    image.fileSize < INQUIRY_IMAGE_MIN_FILE_SIZE ||
    (INQUIRY_IMAGE_MAX_FILE_SIZE > 0 &&
      image.fileSize > INQUIRY_IMAGE_MAX_FILE_SIZE)
  ) {
    return false;
  }

  // mimeType
  if (!image.mimeType?.trim() || !MIME_REGEX.test(image.mimeType)) {
    return false;
  }
  if (
    INQUIRY_IMAGE_ALLOWED_MIME_TYPES.length > 0 &&
    !INQUIRY_IMAGE_ALLOWED_MIME_TYPES.includes(image.mimeType)
  ) {
    return false;
  }

  // width / height
  if (image.width != null && image.width <= 0) return false;
  if (image.height != null && image.height <= 0) return false;

  // createdAt / createdBy
  if (!parseIso(image.createdAt)) return false;
  if (!image.createdBy?.trim()) return false;

  // updatedAt / updatedBy
  if (image.updatedAt != null) {
    const ut = parseIso(image.updatedAt);
    const ct = parseIso(image.createdAt);
    if (!ut || !ct || ut.getTime() < ct.getTime()) return false;
  }
  if (image.updatedBy != null && !image.updatedBy.trim()) return false;

  // deletedAt / deletedBy
  if (image.deletedAt != null) {
    const dt = parseIso(image.deletedAt);
    const ct = parseIso(image.createdAt);
    if (!dt || !ct || dt.getTime() < ct.getTime()) return false;
  }
  if (image.deletedBy != null && !image.deletedBy.trim()) return false;

  return true;
}

/**
 * InquiryImage 集約の簡易バリデーション
 * - id 非空
 * - 各 ImageFile が有効
 * - 各 ImageFile.inquiryId === id
 * - URL 重複なし
 * - MaxImages 制限
 */
export function validateInquiryImage(agg: InquiryImage): boolean {
  if (!agg.id?.trim()) return false;

  if (
    INQUIRY_IMAGE_MAX_IMAGES > 0 &&
    agg.images.length > INQUIRY_IMAGE_MAX_IMAGES
  ) {
    return false;
  }

  const seenUrls = new Set<string>();

  for (const img of agg.images) {
    if (!validateImageFile(img)) return false;
    if (img.inquiryId !== agg.id) return false;

    const u = normalizeUrl(img.fileUrl);
    if (seenUrls.has(u)) return false;
    seenUrls.add(u);
  }

  return true;
}

// ==============================
// Helpers (module private)
// ==============================

function parseIso(s: string | null | undefined): Date | null {
  if (!s) return null;
  const t = Date.parse(s);
  return Number.isNaN(t) ? null : new Date(t);
}

function normalizeUrl(u: string): string {
  return u.trim();
}

function isValidUrl(raw: string): boolean {
  const s = raw.trim();
  if (!s) return false;
  try {
    const u = new URL(s);
    if (!u.protocol || !u.hostname) return false;

    if (INQUIRY_IMAGE_ALLOWED_URL_HOSTS.length > 0) {
      const host = u.hostname.toLowerCase();
      return INQUIRY_IMAGE_ALLOWED_URL_HOSTS.some(
        (allowed) => allowed.toLowerCase() === host
      );
    }

    return true;
  } catch {
    return false;
  }
}
