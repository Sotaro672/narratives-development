// frontend/shell/src/shared/types/fulfillment.ts
// Generated from frontend/order/src/domain/entity/fulfillment.ts
// and backend/internal/domain/fulfillment/entity.go
//
// 本ファイルは Fulfillment 型の共通定義として、フロントエンド間で共有する。

/**
 * FulfillmentStatus
 *
 * backend の仕様:
 * - 空文字でなければ有効とみなされる
 * - ENUM 値はドメイン固有に定義可能
 *
 * 例:
 * "pending" | "processing" | "shipped" | "delivered"
 */
export type FulfillmentStatus = string;

/**
 * Fulfillment
 * backend/internal/domain/fulfillment/entity.go に対応。
 *
 * - createdAt, updatedAt は ISO8601 文字列または Date オブジェクトを許容
 */
export interface Fulfillment {
  id: string;
  orderId: string;
  paymentId: string;
  status: FulfillmentStatus;
  createdAt: string | Date;
  updatedAt: string | Date;
}

/**
 * FulfillmentStatus の妥当性チェック。
 * backend の IsValidStatus と同等: 空文字でなければ有効。
 */
export function isValidFulfillmentStatus(status: FulfillmentStatus): boolean {
  return typeof status === "string" && status.trim().length > 0;
}

/**
 * Fulfillment の簡易バリデーション。
 * - 必須フィールドの非空チェック
 * - 日付のパース可能性と時系列整合チェック
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

/**
 * ISO8601 文字列の簡易検証。
 * Date.parse で解釈可能か確認する。
 */
function isValidIsoDate(v: string | Date): boolean {
  if (v instanceof Date) return !Number.isNaN(v.getTime());
  if (typeof v !== "string" || !v.trim()) return false;
  const t = Date.parse(v);
  return Number.isFinite(t);
}
