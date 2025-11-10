// frontend/order/src/domain/entity/tracking.ts

/**
 * Tracking
 * backend/internal/domain/tracking/entity.go に対応するフロントエンド側エンティティ定義。
 *
 * - 日付は ISO8601 文字列として扱う（UTC 想定）
 * - specialInstructions は任意（null を許容）
 */
export interface Tracking {
  id: string;
  orderId: string;
  trackingNumber: string;
  carrier: string;
  specialInstructions?: string | null;
  createdAt: string;
  updatedAt: string;
}

/**
 * Policy (backend/internal/domain/tracking/entity.go と整合)
 */
export const TRACKING_ID_PREFIX = "";
export const TRACKING_ENFORCE_ID_PREFIX = false;
export const TRACKING_MAX_ID_LENGTH = 128;

export const TRACKING_MIN_TRACKING_NUMBER_LENGTH = 1;
export const TRACKING_MAX_TRACKING_NUMBER_LENGTH = 128;

export const TRACKING_MIN_CARRIER_LENGTH = 1;
export const TRACKING_MAX_CARRIER_LENGTH = 80;

export const TRACKING_MAX_SPECIAL_INSTRUCTIONS_LENGTH = 2000;

// TrackingNumber は英数字・ハイフン・アンダースコア・ドットのみ
export const TRACKING_NUMBER_REGEX = /^[A-Za-z0-9\-_.]+$/;

// 許可キャリア（空配列の場合は全キャリア許可 = Go 実装と同義）
export const TRACKING_ALLOWED_CARRIERS: string[] = [];

/**
 * Tracking エンティティの簡易バリデーション
 * - Go 実装の validate と同等の範囲を TypeScript で確認
 */
export function validateTracking(t: Tracking): boolean {
  // id
  if (!t.id?.trim()) return false;
  if (
    TRACKING_ENFORCE_ID_PREFIX &&
    TRACKING_ID_PREFIX &&
    !t.id.startsWith(TRACKING_ID_PREFIX)
  ) {
    return false;
  }
  if (
    TRACKING_MAX_ID_LENGTH > 0 &&
    [...t.id].length > TRACKING_MAX_ID_LENGTH
  ) {
    return false;
  }

  // orderId
  if (!t.orderId?.trim()) return false;

  // trackingNumber
  const tn = t.trackingNumber?.trim() ?? "";
  const tnLen = [...tn].length;
  if (
    tnLen < TRACKING_MIN_TRACKING_NUMBER_LENGTH ||
    (TRACKING_MAX_TRACKING_NUMBER_LENGTH > 0 &&
      tnLen > TRACKING_MAX_TRACKING_NUMBER_LENGTH)
  ) {
    return false;
  }
  if (!TRACKING_NUMBER_REGEX.test(tn)) return false;

  // carrier
  const carrier = t.carrier?.trim() ?? "";
  const cl = [...carrier].length;
  if (
    cl < TRACKING_MIN_CARRIER_LENGTH ||
    (TRACKING_MAX_CARRIER_LENGTH > 0 && cl > TRACKING_MAX_CARRIER_LENGTH)
  ) {
    return false;
  }
  if (
    TRACKING_ALLOWED_CARRIERS.length > 0 &&
    !TRACKING_ALLOWED_CARRIERS.includes(carrier)
  ) {
    return false;
  }

  // specialInstructions
  if (t.specialInstructions != null) {
    const si = t.specialInstructions.trim();
    if (
      TRACKING_MAX_SPECIAL_INSTRUCTIONS_LENGTH > 0 &&
      [...si].length > TRACKING_MAX_SPECIAL_INSTRUCTIONS_LENGTH
    ) {
      return false;
    }
  }

  // createdAt / updatedAt（最低限のチェック：非空 & 時系列）
  const created = new Date(t.createdAt);
  const updated = new Date(t.updatedAt);
  if (Number.isNaN(created.getTime())) return false;
  if (Number.isNaN(updated.getTime())) return false;
  if (updated.getTime() < created.getTime()) return false;

  return true;
}
