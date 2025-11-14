// frontend/order/src/domain/entity/fulfillment.ts
// Mirror of backend/internal/domain/fulfillment/entity.go
// and source-of-truth for frontend/shell/src/shared/types/fulfillment.ts

/**
 * FulfillmentStatus
 * - backend 側では「非空文字列」であれば有効とみなされるプレースホルダ ENUM。
 * - 必要に応じて実際のステータス値を union 型として拡張してください。
 *
 * 例:
 * export type FulfillmentStatus = "pending" | "processing" | "shipped" | "delivered";
 */
export type FulfillmentStatus = string;

/**
 * Fulfillment
 * - 出荷・配送などのフルフィルメント情報
 * - 日付は ISO8601 (UTC) 文字列で扱うことを推奨
 */
export interface Fulfillment {
  id: string;
  orderId: string;
  paymentId: string;
  status: FulfillmentStatus;
  createdAt: string; // ISO8601
  updatedAt: string; // ISO8601
}

/**
 * ステータス値のバリデーション
 * backend の IsValidStatus(FulfillmentStatus) と同等:
 * - 空文字列でなければ有効
 */
export function isValidFulfillmentStatus(status: FulfillmentStatus): boolean {
  return typeof status === "string" && status.trim().length > 0;
}

/**
 * Fulfillment オブジェクトの簡易バリデーション
 * - 必須フィールドが空でないか
 * - createdAt / updatedAt がパース可能か
 * - status が空文字でないか
 */
export function validateFulfillment(f: Fulfillment): boolean {
  if (!f.id?.trim()) return false;
  if (!f.orderId?.trim()) return false;
  if (!f.paymentId?.trim()) return false;
  if (!isValidFulfillmentStatus(f.status)) return false;

  if (!isValidIsoDate(f.createdAt)) return false;
  if (!isValidIsoDate(f.updatedAt)) return false;

  const created = new Date(f.createdAt).getTime();
  const updated = new Date(f.updatedAt).getTime();
  if (Number.isNaN(created) || Number.isNaN(updated)) return false;
  if (updated < created) return false;

  return true;
}

/** 内部利用: ISO8601 文字列として解釈可能か簡易チェック */
function isValidIsoDate(value: string): boolean {
  if (!value || typeof value !== "string") return false;
  const t = Date.parse(value);
  return Number.isFinite(t);
}
