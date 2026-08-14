// frontend/console/shell/src/features/mint/infrastructure/dto/mintRequestManagementRow.ts

import type { InspectionStatus } from "../../../../shared/types/inspections";
import type { MintStatus } from "../../../../shared/types/mints";

/**
 * GET /mint/requests が返す一覧・詳細表示用DTO。
 *
 * 前提:
 * - Backend BFF responseをそのまま正とする
 * - productions / inspections / mintsのdocIdはすべて同一
 * - productionIdを正とする
 * - id / inspectionId / mintIdのaliasは持たない
 * - mintQuantity / productionQuantityはBackendの必須intに対応する
 * - createdByとrequestedByは相互補完しない
 */
export type MintRequestManagementRowDTO = {
  productionId: string;

  tokenBlueprintId?: string | null;
  tokenName?: string | null;
  productName?: string | null;

  mintQuantity: number;
  productionQuantity: number;

  inspectionStatus?: InspectionStatus | string | null;

  /**
   * Backendが返す親Mintの状態。
   * Frontend側で別のminted booleanへ変換せず、この値を正とする。
   */
  mintStatus?: MintStatus | string | null;

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