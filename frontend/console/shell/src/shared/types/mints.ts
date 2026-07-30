// frontend/console/shell/src/features/mintRequest/domain/mints.ts

/**
 * 親Mintの進行状態。
 * backend/internal/domain/mint/entity.goのMintStatusに対応する。
 */
export type MintStatus =
  | "CREATED"
  | "QUEUED"
  | "MINTING"
  | "PARTIALLY_MINTED"
  | "MINTED"
  | "FAILED_RETRYABLE"
  | "FAILED_FATAL";

/**
 * mintsテーブルの親Mintを表すフロントエンド用の型。
 *
 * backend/internal/domain/mint/entity.goのMint構造体に対応する。
 * 日付フィールドはJSONとの互換性を考慮し、
 * ISO 8601形式の文字列として扱う。
 */
export type Mint = {
  id: string;
  brandId: string;
  tokenBlueprintId: string;
  products: string[];
  status: MintStatus;

  /**
   * Mintドキュメントの作成日時。
   */
  createdAt: string;

  /**
   * Mintドキュメントを作成したmemberId。
   */
  createdBy: string;

  /**
   * Mint申請ボタンを押したmemberId。
   * Mint未申請の場合は未設定。
   */
  requestedBy?: string | null;

  mintedAt?: string | null;
  scheduledBurnDate?: string | null;
};