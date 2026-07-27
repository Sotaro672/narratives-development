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
  /**
   * MintのドキュメントID。
   */
  id: string;

  /**
   * Mintに紐づくブランドID。
   */
  brandId: string;

  /**
   * Mintに紐づくトークン設計ID。
   */
  tokenBlueprintId: string;

  /**
   * inspectionResultがpassedであるproductIdの一覧。
   */
  products: string[];

  /**
   * 親Mintの進行状態。
   */
  status: MintStatus;

  /**
   * 作成日時。
   * ISO 8601形式の文字列を想定する。
   */
  createdAt: string;

  /**
   * 作成者のmemberId。
   */
  createdBy: string;

  /**
   * ミント完了日時。
   * 未完了の場合はnullまたは未設定。
   */
  mintedAt?: string | null;

  /**
   * 焼却予定日時。
   * 未設定の場合はnullまたは未設定。
   */
  scheduledBurnDate?: string | null;
};