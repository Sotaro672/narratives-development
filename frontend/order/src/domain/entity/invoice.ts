// frontend/order/src/domain/entity/invoice.ts
// Generated from backend/internal/domain/invoice/entity.go
// Mirrors web-app/src/shared/types/invoice.ts (TS をソース・オブ・トゥルースと想定)

/**
 * InvoiceStatus
 * backend/internal/domain/invoice/entity.go の InvoiceStatus に対応。
 *
 * - "unpaid"   : 未払い
 * - "paid"     : 支払済み
 * - "refunded" : 返金済み
 */
export type InvoiceStatus = "unpaid" | "paid" | "refunded";

/** InvoiceStatus の妥当性チェック */
export function isValidInvoiceStatus(s: string): s is InvoiceStatus {
  return s === "unpaid" || s === "paid" || s === "refunded";
}

/**
 * OrderItemInvoice
 * backend/internal/domain/invoice/entity.go の OrderItemInvoice に対応。
 *
 * - 金額は整数（最小 0、MaxMoney が 0 の場合は上限なし）
 * - createdAt / updatedAt は ISO8601 (UTC) 文字列で扱う
 */
export interface OrderItemInvoice {
  id: string;
  orderItemId: string;
  unitPrice: number;
  totalPrice: number;
  createdAt: string; // ISO8601
  updatedAt: string; // ISO8601
}

/**
 * Invoice
 * backend/internal/domain/invoice/entity.go の Invoice に対応。
 *
 * - orderItemInvoices: OrderItemInvoice の配列
 * - subtotal, discountAmount, taxAmount, shippingCost, totalAmount は整数
 * - currency は 3文字の通貨コード (例: "JPY")
 * - createdAt / updatedAt は ISO8601 (UTC) 文字列
 */
export interface Invoice {
  orderId: string;
  orderItemInvoices: OrderItemInvoice[];
  subtotal: number;
  discountAmount: number;
  taxAmount: number;
  shippingCost: number;
  totalAmount: number;
  currency: string;
  createdAt: string; // ISO8601
  updatedAt: string; // ISO8601
  billingAddressId: string;
}

/**
 * Policy
 * backend/internal/domain/invoice/entity.go の Policy と整合
 *
 * - MinMoney: 各金額の下限
 * - MaxMoney: 0 の場合、上限チェック無効
 * - EnforceTotalEquality:
 *   totalAmount === subtotal - discountAmount + taxAmount + shippingCost を要求
 * - CURRENCY_CODE_REGEX: /^[A-Z]{3}$/ にマッチ
 */
export const MIN_MONEY = 0;
export const MAX_MONEY = 0; // 0 -> no upper bound

export const ENFORCE_TOTAL_EQUALITY = true;

export const CURRENCY_CODE_REGEX = /^[A-Z]{3}$/;

/** 内部ヘルパー: 金額バリデーション（Min/Max ポリシー適用） */
function moneyOK(v: number): boolean {
  if (!Number.isFinite(v) || !Number.isInteger(v)) return false;
  if (v < MIN_MONEY) return false;
  if (MAX_MONEY > 0 && v > MAX_MONEY) return false;
  return true;
}

/** ISO8601 日付チェック（軽量版） */
function isValidISODateString(v: string | null | undefined): boolean {
  if (!v || !v.trim()) return false;
  const t = Date.parse(v);
  return !Number.isNaN(t);
}

/**
 * OrderItemInvoice の簡易バリデーション
 * backend の OrderItemInvoice.validate() に概ね追従
 */
export function validateOrderItemInvoice(oi: OrderItemInvoice): boolean {
  if (!oi.id?.trim()) return false;
  if (!oi.orderItemId?.trim()) return false;

  if (!moneyOK(oi.unitPrice)) return false;
  if (!moneyOK(oi.totalPrice)) return false;

  if (!isValidISODateString(oi.createdAt)) return false;
  if (!isValidISODateString(oi.updatedAt)) return false;

  const created = Date.parse(oi.createdAt);
  const updated = Date.parse(oi.updatedAt);
  if (updated < created) return false;

  return true;
}

/**
 * Invoice の簡易バリデーション
 * backend の Invoice.validate() と整合する範囲で実装。
 */
export function validateInvoice(inv: Invoice): boolean {
  // orderId
  if (!inv.orderId?.trim()) return false;

  // billingAddressId
  if (!inv.billingAddressId?.trim()) return false;

  // currency
  const cur = inv.currency?.toUpperCase().trim();
  if (!cur || !CURRENCY_CODE_REGEX.test(cur)) return false;

  // createdAt / updatedAt
  if (!isValidISODateString(inv.createdAt)) return false;
  if (!isValidISODateString(inv.updatedAt)) return false;

  const created = Date.parse(inv.createdAt);
  const updated = Date.parse(inv.updatedAt);
  if (updated < created) return false;

  // amounts
  if (
    !moneyOK(inv.subtotal) ||
    !moneyOK(inv.discountAmount) ||
    !moneyOK(inv.taxAmount) ||
    !moneyOK(inv.shippingCost) ||
    !moneyOK(inv.totalAmount)
  ) {
    return false;
  }

  // totalAmount consistency
  if (ENFORCE_TOTAL_EQUALITY) {
    const computed =
      inv.subtotal - inv.discountAmount + inv.taxAmount + inv.shippingCost;
    if (inv.totalAmount !== computed) {
      return false;
    }
  }

  // orderItemInvoices
  for (const oi of inv.orderItemInvoices) {
    if (!validateOrderItemInvoice(oi)) return false;
  }

  return true;
}

/**
 * 合計金額を再計算（TS 側ユーティリティ）
 * backend の ComputeTotal() と同等。
 */
export function computeInvoiceTotal(inv: Pick<
  Invoice,
  "subtotal" | "discountAmount" | "taxAmount" | "shippingCost"
>): number {
  return (
    inv.subtotal - inv.discountAmount + inv.taxAmount + inv.shippingCost
  );
}
