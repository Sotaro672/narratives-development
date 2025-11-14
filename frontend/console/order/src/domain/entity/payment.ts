// frontend/order/src/domain/entity/payment.ts
// Mirror of backend/internal/domain/payment/entity.go
// Source of truth for Payment 型は本ファイルおよび
// frontend/shell/src/shared/types/payment.ts（後続で作成/同期）とする。

/**
 * PaymentStatus
 * backend の PaymentStatus と同様に「任意の非空文字列」を許容するが、
 * AllowedStatuses を指定した場合はそのホワイトリストに制限される。
 */
export type PaymentStatus = string;

// Optional policy: 空でなければ任意許容。値を追加するとその一覧に制限される。
export const AllowedStatuses: ReadonlySet<PaymentStatus> = new Set<PaymentStatus>([
  // 例:
  // "authorized",
  // "captured",
  // "failed",
  // "refunded",
]);

/**
 * backend の IsValidStatus と同じロジック:
 * - 空文字は不可
 * - AllowedStatuses が空なら「任意の非空文字列」を許容
 * - AllowedStatuses に値があれば、その中に含まれる値のみ許容
 */
export function isValidPaymentStatus(status: PaymentStatus): boolean {
  if (!status || !status.trim()) return false;
  if (AllowedStatuses.size === 0) return true;
  return AllowedStatuses.has(status);
}

/**
 * Payment (Entity)
 * backend/internal/domain/payment/entity.go の Payment 構造体に対応。
 *
 * - createdAt / updatedAt / deletedAt は ISO8601 文字列として扱うことを推奨
 *   (ドメイン層では Date 型または string 型で運用可能)
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
 * backend の validate() と概ね整合するチェック。
 * - 空文字チェック
 * - 金額の下限 (MinAmount = 0)
 * - Status の妥当性 (isValidPaymentStatus)
 * - 日付フォーマットは「非空であること」のみ（詳細検証は必要に応じて呼び出し側で実施）
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

  // deletedAt が存在する場合の詳細な時系列チェックは省略（必要に応じて拡張）
  return true;
}
