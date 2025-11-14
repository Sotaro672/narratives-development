// frontend/shell/src/shared/types/payment.ts
// Generated from frontend/order/src/domain/entity/payment.ts
// and backend/internal/domain/payment/entity.go
//
// 本ファイルを Payment 型のソース・オブ・トゥルースとし、
// 各フロントエンド機能・モックデータはこの定義に準拠させる。

/**
 * PaymentStatus
 *
 * backend の仕様と同様:
 * - 空文字は不可
 * - 許可リスト (ALLOWED_PAYMENT_STATUSES) が空の場合:
 *     任意の非空文字列を有効とみなす
 * - 許可リストが設定されている場合:
 *     その中に含まれる値のみ有効
 */
export type PaymentStatus = string;

/**
 * 許可するステータス一覧。
 * 空のままにしておけば「任意の非空文字列」を許容する挙動になる。
 * 必要になったらここに `"authorized" | "captured" | "failed" | ...` 等を追加する。
 */
export const ALLOWED_PAYMENT_STATUSES: ReadonlySet<PaymentStatus> =
  new Set<PaymentStatus>([
    // 例:
    // "authorized",
    // "captured",
    // "failed",
    // "refunded",
  ]);

/**
 * backend の IsValidStatus と同じロジック。
 */
export function isValidPaymentStatus(status: PaymentStatus): boolean {
  if (!status || !status.trim()) return false;
  if (ALLOWED_PAYMENT_STATUSES.size === 0) return true;
  return ALLOWED_PAYMENT_STATUSES.has(status);
}

/**
 * Payment
 * backend/internal/domain/payment/entity.go の Payment を TS 化した共通型。
 *
 * - createdAt / updatedAt / deletedAt:
 *     API 通信では ISO8601 文字列、アプリ内では Date に変換して扱う想定のため union。
 */
export interface Payment {
  id: string;
  invoiceId: string;
  billingAddressId: string;
  amount: number;
  status: PaymentStatus;
  errorType?: string | null;
  createdAt: string | Date;
  updatedAt: string | Date;
  deletedAt?: string | Date | null;
}

/**
 * 簡易バリデーション
 * backend の validate() と概ね整合する範囲で実装。
 */
export function validatePayment(p: Payment): boolean {
  if (!p.id?.trim()) return false;
  if (!p.invoiceId?.trim()) return false;
  if (!p.billingAddressId?.trim()) return false;

  if (typeof p.amount !== "number" || p.amount < 0) {
    return false;
  }

  if (!isValidPaymentStatus(p.status)) {
    return false;
  }

  if (p.errorType != null && !p.errorType.trim()) {
    return false;
  }

  if (!p.createdAt) return false;
  if (!p.updatedAt) return false;

  // deletedAt は存在する場合のみ、詳細チェックは呼び出し側に委譲
  return true;
}
