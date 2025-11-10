// frontend/order/src/domain/entity/transfer.ts

/**
 * TransferStatus
 * backend/internal/domain/transfer/entity.go に準拠
 */
export type TransferStatus = "fulfilled" | "requested" | "error";

/**
 * TransferErrorType
 * backend/internal/domain/transfer/entity.go に準拠
 */
export type TransferErrorType =
  | "insufficient_balance"
  | "invalid_address"
  | "network_error"
  | "timeout"
  | "unknown";

/**
 * Transfer
 * web-app/src/shared/types/transfer.ts 相当のフロントエンド用エンティティ定義
 */
export interface Transfer {
  id: string;
  mintAddress: string;
  fromAddress: string;
  toAddress: string;
  requestedAt: string; // ISO8601
  transferredAt?: string | null; // ISO8601 or null
  status: TransferStatus;
  errorType?: TransferErrorType | null;
}

/**
 * Base58 / ポリシー設定
 * backend の定義と整合
 */
export const TRANSFER_BASE58_MIN_LEN = 32;
export const TRANSFER_BASE58_MAX_LEN = 44;
export const TRANSFER_BASE58_ALPHABET =
  "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

/**
 * ステータス遷移許可マトリクス
 * backend/internal/domain/transfer/entity.go の allowedTransitions と同期
 */
const allowedTransitions: Record<
  TransferStatus,
  Partial<Record<TransferStatus, true>>
> = {
  requested: { fulfilled: true, error: true },
  error: { requested: true },
  fulfilled: {},
};

/**
 * Status / ErrorType バリデーション
 */
export function isValidTransferStatus(s: string): s is TransferStatus {
  return s === "fulfilled" || s === "requested" || s === "error";
}

export function isValidTransferErrorType(
  t: string | null | undefined
): t is TransferErrorType {
  if (!t) return false;
  return (
    t === "insufficient_balance" ||
    t === "invalid_address" ||
    t === "network_error" ||
    t === "timeout" ||
    t === "unknown"
  );
}

/**
 * Base58 アドレス簡易検証（backend と整合する範囲）
 */
export function isValidBase58Address(addr: string): boolean {
  const s = addr.trim();
  if (!s) return false;
  const len = [...s].length;
  if (len < TRANSFER_BASE58_MIN_LEN) return false;
  if (TRANSFER_BASE58_MAX_LEN > 0 && len > TRANSFER_BASE58_MAX_LEN) {
    return false;
  }
  for (const ch of s) {
    if (!TRANSFER_BASE58_ALPHABET.includes(ch)) {
      return false;
    }
  }
  return true;
}

/**
 * ステータス遷移が許可されているかチェック
 */
export function isTransferTransitionAllowed(
  from: TransferStatus,
  to: TransferStatus
): boolean {
  const nexts = allowedTransitions[from];
  return !!nexts && !!nexts[to];
}

/**
 * Transfer の整合性チェック
 * backend の validate() ロジックに沿った簡易版。
 */
export function validateTransfer(t: Transfer): boolean {
  // id
  if (!t.id.trim()) return false;

  // addresses
  if (!isValidBase58Address(t.mintAddress)) return false;
  if (!isValidBase58Address(t.fromAddress)) return false;
  if (!isValidBase58Address(t.toAddress)) return false;

  // requestedAt
  const requestedAt = new Date(t.requestedAt);
  if (Number.isNaN(requestedAt.getTime())) return false;

  // status
  if (!isValidTransferStatus(t.status)) return false;

  // transferredAt
  const transferred =
    t.transferredAt == null || t.transferredAt === ""
      ? null
      : new Date(t.transferredAt);
  if (transferred && Number.isNaN(transferred.getTime())) return false;

  // errorType（空文字は型上発生しないので null/undefined のみ考慮）
  const errorType = t.errorType ?? null;

  // 状態ごとの整合性
  switch (t.status) {
    case "requested":
      if (errorType !== null) return false;
      if (transferred !== null) return false;
      break;

    case "fulfilled":
      if (errorType !== null) return false;
      if (!transferred) return false;
      if (transferred.getTime() < requestedAt.getTime()) return false;
      break;

    case "error":
      if (!errorType || !isValidTransferErrorType(errorType)) return false;
      if (transferred !== null) return false;
      break;
  }

  return true;
}
