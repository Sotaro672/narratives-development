// frontend/console/shell/src/features/mintRequest/application/dto/mintRequestManagementRow.ts

import type { InspectionStatus } from "../../../../shared/types/inspections";

/**
 * MintRequestQueryServiceが返す一覧用DTO。
 *
 * 前提:
 * - productions / inspections / mintsのdocIdはすべて同一
 * - productionIdを正とする
 * - id / inspectionId / mintIdは主キーとして扱わない
 * - createdByとrequestedByは相互補完しない
 */
export type MintRequestManagementRowDTO = {
  productionId: string;

  tokenBlueprintId?: string | null;
  tokenName?: string | null;
  productName?: string | null;

  mintQuantity?: number | null;
  productionQuantity?: number | null;

  inspectionStatus?: InspectionStatus | string | null;

  /**
   * mintsドキュメントを作成したmemberId。
   */
  createdBy?: string | null;

  /**
   * createdByに対応する表示名。
   */
  createdByName?: string | null;

  /**
   * Mint申請ボタンを押したmemberId。
   */
  requestedBy?: string | null;

  /**
   * requestedByに対応する表示名。
   */
  requestedByName?: string | null;

  mintedAt?: string | null;

  /**
   * 一覧画面側で算出または補完される状態。
   */
  minted?: boolean | null;

  /**
   * 現行の詳細画面との互換用項目。
   * 一覧APIから返らない場合はnullとして扱う。
   */
  productBlueprintId?: string | null;
  scheduledBurnDate?: string | null;
};