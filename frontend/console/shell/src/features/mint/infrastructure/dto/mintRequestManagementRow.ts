// frontend/console/shell/src/features/mintRequest/application/dto/mintRequestManagementRow.ts

import type { InspectionStatus } from "../../../../shared/types/inspections";
import type { MintStatus } from "../../../../shared/types/mints";

/**
 * MintRequestQueryServiceが返す一覧・詳細表示用DTO。
 *
 * 前提:
 * - productions / inspections / mintsのdocIdはすべて同一
 * - productionIdを正とする
 * - id / inspectionId / mintIdは主キーとして扱わない
 * - createdByとrequestedByは相互補完しない
 * - BackendのGET /mint/requests responseをそのまま正とする
 */
export type MintRequestManagementRowDTO = {
  productionId: string;

  tokenBlueprintId?: string | null;
  tokenName?: string | null;
  productName?: string | null;

  mintQuantity?: number | null;
  productionQuantity?: number | null;

  inspectionStatus?:
    | InspectionStatus
    | string
    | null;

  /**
   * Backendが返す親Mintの状態。
   */
  mintStatus?:
    | MintStatus
    | string
    | null;

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
};