// frontend/ad/src/domain/entity/campaignImage.ts

/**
 * CampaignImage domain entity (frontend)
 * backend/internal/domain/campaignImage/entity.go に対応。
 *
 * - SQL/Go 側は CampaignImage を単独エンティティとして管理
 * - フロントエンドでは API / MFE 間で共有される型として利用する想定
 */

/**
 * Default GCS bucket for CampaignImage objects.
 * backend の DefaultBucket と同期。
 */
export const CAMPAIGN_IMAGE_DEFAULT_BUCKET =
  "narratives_development_campaign_image";

/**
 * GCSDeleteOp
 * GCS 上のオブジェクト削除指示用。
 */
export interface GCSDeleteOp {
  bucket: string;
  objectPath: string;
}

/**
 * CampaignImage
 * backend/internal/domain/campaignImage/entity.go の構造に対応。
 *
 * - width / height / fileSize / mimeType は任意
 * - imageUrl は必須の有効な URL
 */
export interface CampaignImage {
  id: string;
  campaignId: string;
  imageUrl: string;
  width?: number;
  height?: number;
  fileSize?: number;
  mimeType?: string;
}

/**
 * Policy (backend と同期)
 */
export const CAMPAIGN_IMAGE_REQUIRE_POSITIVE_DIMENSIONS = true;

export const CAMPAIGN_IMAGE_MIN_FILE_SIZE = 1; // bytes
export const CAMPAIGN_IMAGE_MAX_FILE_SIZE = 20 * 1024 * 1024; // 20MB

export const CAMPAIGN_IMAGE_ALLOWED_MIME_TYPES: string[] = [
  "image/jpeg",
  "image/png",
  "image/webp",
  "image/gif",
];

// 空配列の場合は全ホスト許可（Go 実装と同じ意味）
export const CAMPAIGN_IMAGE_ALLOWED_URL_HOSTS: string[] = [];

// MIME フォーマットチェック用
const MIME_REGEX = /^[a-zA-Z0-9.+-]+\/[a-zA-Z0-9.+-]+$/;

/**
 * CampaignImage の簡易バリデーション
 * backend の validate() / fileSizeOK / mimeOK / urlOK と整合する範囲で実装。
 */
export function validateCampaignImage(c: CampaignImage): boolean {
  // id
  if (!c.id?.trim()) return false;

  // campaignId
  if (!c.campaignId?.trim()) return false;

  // imageUrl
  if (!isValidUrl(c.imageUrl)) return false;

  // width / height
  if (c.width != null) {
    if (!Number.isFinite(c.width)) return false;
    if (CAMPAIGN_IMAGE_REQUIRE_POSITIVE_DIMENSIONS && c.width <= 0) {
      return false;
    }
  }
  if (c.height != null) {
    if (!Number.isFinite(c.height)) return false;
    if (CAMPAIGN_IMAGE_REQUIRE_POSITIVE_DIMENSIONS && c.height <= 0) {
      return false;
    }
  }

  // fileSize
  if (c.fileSize != null) {
    if (!Number.isFinite(c.fileSize)) return false;
    if (c.fileSize < CAMPAIGN_IMAGE_MIN_FILE_SIZE) return false;
    if (
      CAMPAIGN_IMAGE_MAX_FILE_SIZE > 0 &&
      c.fileSize > CAMPAIGN_IMAGE_MAX_FILE_SIZE
    ) {
      return false;
    }
  }

  // mimeType
  if (c.mimeType != null) {
    const mt = c.mimeType.trim();
    if (!mt || !MIME_REGEX.test(mt)) return false;
    if (
      CAMPAIGN_IMAGE_ALLOWED_MIME_TYPES.length > 0 &&
      !CAMPAIGN_IMAGE_ALLOWED_MIME_TYPES.includes(mt)
    ) {
      return false;
    }
  }

  return true;
}

// ==============================
// Helpers
// ==============================

function isValidUrl(raw: string): boolean {
  const s = raw.trim();
  if (!s) return false;
  try {
    const u = new URL(s);
    if (!u.protocol || !u.hostname) return false;

    if (CAMPAIGN_IMAGE_ALLOWED_URL_HOSTS.length > 0) {
      const host = u.hostname.toLowerCase();
      return CAMPAIGN_IMAGE_ALLOWED_URL_HOSTS.some(
        (allowed) => allowed.toLowerCase() === host
      );
    }

    return true;
  } catch {
    return false;
  }
}
