// frontend/shell/src/shared/types/invoice.ts
// (Generated from frontend/order/src/domain/entity/invoice.ts
//  and backend/internal/domain/invoice/entity.go)

/**
 * InvoiceStatus
 * 支払い状態を示す列挙。
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
 * 注文明細単位の請求情報。
 * backend/internal/domain/invoice/entity.go の OrderItemInvoice に対応。
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
 * 請求書データの基本構造。
 * backend/internal/domain/invoice/entity.go の Invoice に対応。
 */
export interface Invoice {
  orderId: string;
  orderItemInvoices: OrderItemInvoice[];
  subtotal: number;
  discountAmount: number;
  taxAmount: number;
  shippingCost: number;
  totalAmount: number;
  currency: string; // ISO 4217 形式の3文字コード (例: "JPY")
  createdAt: string; // ISO8601
  updatedAt: string; // ISO8601
  billingAddressId: string;
}

/**
 * Policy 定数（backend と同期）
 */
export const INVOICE_POLICY = {
  minMoney: 0,
  maxMoney: 0, // 0 → 上限なし
  enforceTotalEquality: true,
  currencyCodeRegex: /^[A-Z]{3}$/,
};

/**
 * 金額バリデーション関数
 * backend の moneyOK() に対応。
 */
export function moneyOK(v: number): boolean {
  const { minMoney, maxMoney } = INVOICE_POLICY;
  if (!Number.isFinite(v) || !Number.isInteger(v)) return false;
  if (v < minMoney) return false;
  if (maxMoney > 0 && v > maxMoney) return false;
  return true;
}

/**
 * OrderItemInvoice のバリデーション
 */
export function validateOrderItemInvoice(oi: OrderItemInvoice): boolean {
  if (!oi.id?.trim()) return false;
  if (!oi.orderItemId?.trim()) return false;
  if (!moneyOK(oi.unitPrice) || !moneyOK(oi.totalPrice)) return false;
  if (!oi.createdAt || !oi.updatedAt) return false;
  const created = Date.parse(oi.createdAt);
  const updated = Date.parse(oi.updatedAt);
  if (isNaN(created) || isNaN(updated)) return false;
  if (updated < created) return false;
  return true;
}

/**
 * Invoice のバリデーション
 * backend の validate() ロジックに基づく。
 */
export function validateInvoice(inv: Invoice): boolean {
  const { enforceTotalEquality, currencyCodeRegex } = INVOICE_POLICY;

  if (!inv.orderId?.trim()) return false;
  if (!inv.billingAddressId?.trim()) return false;

  // 通貨チェック
  const currency = inv.currency?.toUpperCase().trim();
  if (!currency || !currencyCodeRegex.test(currency)) return false;

  // 日付チェック
  const created = Date.parse(inv.createdAt);
  const updated = Date.parse(inv.updatedAt);
  if (isNaN(created) || isNaN(updated)) return false;
  if (updated < created) return false;

  // 金額チェック
  if (
    !moneyOK(inv.subtotal) ||
    !moneyOK(inv.discountAmount) ||
    !moneyOK(inv.taxAmount) ||
    !moneyOK(inv.shippingCost) ||
    !moneyOK(inv.totalAmount)
  ) {
    return false;
  }

  // 合計一致チェック
  if (enforceTotalEquality) {
    const expected =
      inv.subtotal - inv.discountAmount + inv.taxAmount + inv.shippingCost;
    if (inv.totalAmount !== expected) return false;
  }

  // 注文明細チェック
  for (const oi of inv.orderItemInvoices) {
    if (!validateOrderItemInvoice(oi)) return false;
  }

  return true;
}

/**
 * 合計金額を再計算するユーティリティ
 */
export function computeInvoiceTotal(inv: Pick<
  Invoice,
  "subtotal" | "discountAmount" | "taxAmount" | "shippingCost"
>): number {
  return (
    inv.subtotal - inv.discountAmount + inv.taxAmount + inv.shippingCost
  );
}
