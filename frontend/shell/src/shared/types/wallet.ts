// frontend/shell/src/shared/types/wallet.ts
// ------------------------------------------------------
// Shared Wallet types
// Mirrors frontend/inquiry/src/domain/entity/wallet.ts
// and backend/internal/domain/wallet/entity.go
// ------------------------------------------------------

/**
 * WalletStatus
 * 'active' | 'inactive'
 */
export type WalletStatus = "active" | "inactive";

/**
 * Wallet
 *
 * NOTE:
 * - In shared types,日時フィールドは `Date | string` とする。
 *   （APIレスポンス文字列 / クライアント内Date両対応）
 */
export interface Wallet {
  walletAddress: string;
  tokens: string[];
  lastUpdatedAt: Date | string;
  status: WalletStatus;
  createdAt: Date | string;
  updatedAt: Date | string;
}
