// frontend/shell/src/shared/types/transaction.ts

/**
 * TransactionType
 * backend/internal/domain/transaction/entity.go の TransactionType に対応。
 *
 * - "receive" : 受取
 * - "send"    : 送金
 */
export type TransactionType = "receive" | "send";

/** TransactionType の妥当性チェック */
export function isValidTransactionType(t: string): t is TransactionType {
  return t === "receive" || t === "send";
}

/**
 * Transaction
 * backend/internal/domain/transaction/entity.go の Transaction に対応。
 *
 * - timestamp は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）を想定
 * - description は空文字列も許容
 */
export interface Transaction {
  id: string;
  accountId: string;
  brandName: string;
  type: TransactionType;
  amount: number;
  currency: string; // 3文字の通貨コード (例: "JPY", "USD")
  fromAccount: string;
  toAccount: string;
  timestamp: string;
  description: string;
}

/**
 * Policy
 * backend/internal/domain/transaction/entity.go の Policy と整合する定数群。
 */
export const MIN_TRANSACTION_AMOUNT = 0;
export const MAX_TRANSACTION_AMOUNT = 0; // 0 -> 上限なし
export const CURRENCY_CODE_REGEX = /^[A-Z]{3}$/;

/**
 * 通貨許可リスト。
 * - 空配列: Regexを満たす通貨コードを全許可
 * - 要素あり: その配列に含まれる通貨コードのみ許可
 */
export const ALLOWED_CURRENCIES: string[] = [];

/**
 * Transaction の簡易バリデーション
 * backend の validate() ロジックと可能な範囲で整合。
 */
export function validateTransaction(tx: Transaction): boolean {
  // id
  if (!tx.id?.trim()) return false;

  // accountId
  if (!tx.accountId?.trim()) return false;

  // brandName
  if (!tx.brandName?.trim()) return false;

  // type
  if (!isValidTransactionType(tx.type)) return false;

  // amount
  if (
    typeof tx.amount !== "number" ||
    !Number.isFinite(tx.amount) ||
    !Number.isInteger(tx.amount) ||
    tx.amount < MIN_TRANSACTION_AMOUNT ||
    (MAX_TRANSACTION_AMOUNT > 0 && tx.amount > MAX_TRANSACTION_AMOUNT)
  ) {
    return false;
  }

  // currency
  const cur = tx.currency?.toUpperCase().trim();
  if (!cur || !CURRENCY_CODE_REGEX.test(cur)) return false;
  if (ALLOWED_CURRENCIES.length > 0 && !ALLOWED_CURRENCIES.includes(cur)) {
    return false;
  }

  // fromAccount / toAccount
  if (!tx.fromAccount?.trim()) return false;
  if (!tx.toAccount?.trim()) return false;

  // timestamp
  if (!tx.timestamp || Number.isNaN(Date.parse(tx.timestamp))) return false;

  // description: 空文字列は許容（必須ではない）

  return true;
}

/**
 * GraphQL / API 通信用 DTO
 * - 作成・取得時の型合わせ用
 */
export interface TransactionInput {
  id?: string;
  accountId: string;
  brandName: string;
  type: TransactionType;
  amount: number;
  currency: string;
  fromAccount: string;
  toAccount: string;
  timestamp: string; // ISO8601形式
  description?: string;
}
