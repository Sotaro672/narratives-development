// frontend/shell/src/shared/types/transfer.ts

/**
 * TransferStatus
 * backend/internal/domain/transfer/entity.go に準拠。
 *
 * - "fulfilled" : 転送完了
 * - "requested" : 転送要求中
 * - "error"     : エラー発生
 */
export type TransferStatus = "fulfilled" | "requested" | "error";

/**
 * TransferErrorType
 * backend/internal/domain/transfer/entity.go に準拠。
 *
 * - "insufficient_balance" : 残高不足
 * - "invalid_address"      : 無効な宛先
 * - "network_error"        : ネットワークエラー
 * - "timeout"              : タイムアウト
 * - "unknown"              : 不明なエラー
 */
export type TransferErrorType =
  | "insufficient_balance"
  | "invalid_address"
  | "network_error"
  | "timeout"
  | "unknown";

/**
 * Transfer
 * web-app/src/shared/types/transfer.ts 相当。
 * NFTトークン送信やブロックチェーン転送の状態を表す。
 */
export interface Transfer {
  id: string;
  mintAddress: string;
  fromAddress: string;
  toAddress: string;
  requestedAt: string; // ISO8601 (例: "2025-01-10T00:00:00Z")
  transferredAt?: string | null; // ISO8601 or null
  status: TransferStatus;
  errorType?: TransferErrorType | null;
}

/**
 * TransferStatus 表示ラベル
 */
export function getTransferStatusLabel(status: TransferStatus): string {
  switch (status) {
    case "fulfilled":
      return "転送完了";
    case "requested":
      return "転送要求中";
    case "error":
      return "エラー";
    default:
      return "不明";
  }
}

/**
 * TransferErrorType 表示ラベル
 */
export function getTransferErrorLabel(error?: TransferErrorType | null): string {
  if (!error) return "—";
  switch (error) {
    case "insufficient_balance":
      return "残高不足";
    case "invalid_address":
      return "宛先不正";
    case "network_error":
      return "ネットワークエラー";
    case "timeout":
      return "タイムアウト";
    case "unknown":
      return "不明エラー";
    default:
      return "—";
  }
}
