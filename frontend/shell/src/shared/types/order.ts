// frontend/shell/src/shared/types/order.ts

/**
 * LegacyOrderStatus
 * backend/internal/domain/order/entity.go の LegacyOrderStatus に対応。
 *
 * - "paid"
 * - "transferred"
 */
export type LegacyOrderStatus = "paid" | "transferred";

/** LegacyOrderStatus の妥当性チェック */
export function isValidLegacyStatus(s: string): s is LegacyOrderStatus {
  return s === "paid" || s === "transferred";
}

/**
 * Order
 * backend/internal/domain/order/entity.go および
 * frontend/order/src/domain/entity/order.ts の Order に対応。
 *
 * - 日付は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）を想定
 * - items は orderItem の ID 配列
 * - trackingId, transfferedDate, updatedBy, deletedAt, deletedBy は任意
 * - プロパティ名は backend 実装に合わせて `transfferedDate`（綴り注意）
 */
export interface Order {
  id: string;
  orderNumber: string;
  status: LegacyOrderStatus;
  userId: string;
  shippingAddressId: string;
  billingAddressId: string;
  listId: string;
  items: string[];
  invoiceId: string;
  paymentId: string;
  fulfillmentId: string;
  trackingId?: string | null;
  transfferedDate?: string | null;
  createdAt: string;
  updatedAt: string;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * OrderPatch
 * backend/internal/domain/order/entity.go の OrderPatch に対応。
 * - `undefined` / `null` は「変更なし」を表現する用途を想定。
 */
export interface OrderPatch {
  orderNumber?: string | null;
  status?: LegacyOrderStatus | null;
  userId?: string | null;
  shippingAddressId?: string | null;
  billingAddressId?: string | null;
  listId?: string | null;
  items?: string[] | null;
  invoiceId?: string | null;
  paymentId?: string | null;
  fulfillmentId?: string | null;
  trackingId?: string | null;
  transfferedDate?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * Policy
 * backend/internal/domain/order/entity.go の Policy に対応。
 */
export const ORDER_NUMBER_REGEX = /^[A-Z0-9\-]{1,32}$/;
export const MIN_ITEMS_REQUIRED = 1;

/**
 * Order の簡易バリデーション
 * backend の validate() ロジックと整合する範囲でフロント側チェックを行う。
 */
export function validateOrder(o: Order): boolean {
  // id
  if (!o.id || !o.id.trim()) return false;

  // orderNumber
  if (!o.orderNumber || !o.orderNumber.trim()) return false;
  if (!ORDER_NUMBER_REGEX.test(o.orderNumber)) return false;

  // status
  if (!isValidLegacyStatus(o.status)) return false;

  // required ids
  if (!o.userId?.trim()) return false;
  if (!o.shippingAddressId?.trim()) return false;
  if (!o.billingAddressId?.trim()) return false;
  if (!o.listId?.trim()) return false;

  // items
  if (!Array.isArray(o.items) || o.items.length < MIN_ITEMS_REQUIRED) {
    return false;
  }
  const seen = new Set<string>();
  for (const it of o.items) {
    const v = (it ?? "").trim();
    if (!v || seen.has(v)) return false;
    seen.add(v);
  }

  // invoice / payment / fulfillment
  if (!o.invoiceId?.trim()) return false;
  if (!o.paymentId?.trim()) return false;
  if (!o.fulfillmentId?.trim()) return false;

  // trackingId (任意だが、指定する場合は非空)
  if (o.trackingId !== undefined && o.trackingId !== null) {
    if (!o.trackingId.trim()) return false;
  }

  // createdAt / updatedAt
  const created = parseIso(o.createdAt);
  const updated = parseIso(o.updatedAt);
  if (!created || !updated) return false;
  if (updated.getTime() < created.getTime()) return false;

  // transfferedDate: 任意だが、ある場合は createdAt 以降
  if (o.transfferedDate != null && o.transfferedDate !== "") {
    const td = parseIso(o.transfferedDate);
    if (!td || td.getTime() < created.getTime()) return false;
  }

  // updatedBy: 任意だが、ある場合は非空
  if (o.updatedBy != null && o.updatedBy !== "" && !o.updatedBy.trim()) {
    return false;
  }

  // deletedAt / deletedBy の整合性
  const hasDeletedAt =
    o.deletedAt != null && String(o.deletedAt).trim() !== "";
  const hasDeletedBy =
    o.deletedBy != null && String(o.deletedBy).trim() !== "";

  if (hasDeletedAt !== hasDeletedBy) {
    return false;
  }

  if (hasDeletedAt && hasDeletedBy) {
    const da = parseIso(String(o.deletedAt));
    if (!da) return false;
    if (da.getTime() < created.getTime()) return false;
    if (!String(o.deletedBy).trim()) return false;
  }

  return true;
}

/** ISO8601 文字列を Date に変換（失敗時は null） */
function parseIso(s: string | null | undefined): Date | null {
  if (!s) return null;
  const t = Date.parse(s);
  if (Number.isNaN(t)) return null;
  return new Date(t);
}
