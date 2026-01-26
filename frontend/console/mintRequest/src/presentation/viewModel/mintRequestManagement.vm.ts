// frontend/console/mintRequest/src/presentation/viewModel/mintRequestManagement.vm.ts

import type { InspectionStatus } from "../../domain/entity/inspections";

// ============================================================
// ViewModel Types for MintRequestManagement (List Screen)
// ============================================================

export type MintRequestRowStatus = "planning" | "requested" | "minted";

/**
 * 画面で使う “一覧 1 行分” の ViewModel
 * - id は productionId (= inspectionId 扱い)
 */
export type MintRequestManagementRowVM = {
  id: string;

  tokenName: string | null;
  productName: string | null;

  mintQuantity: number;
  productionQuantity: number;

  status: MintRequestRowStatus;
  inspectionStatus: InspectionStatus;

  createdByName: string | null;
  mintedAt: string | null;

  // detail 画面や更新用に保持（表示には使わない前提でも、payload 構築で必要）
  tokenBlueprintId: string | null;
  requestedBy: string | null;

  productBlueprintId: string | null;
  scheduledBurnDate: string | null;
  minted: boolean;

  statusLabel: string;
};

/**
 * 一覧全体で使う VM（必要な場合のみ利用）
 */
export type MintRequestManagementVM = {
  rows: MintRequestManagementRowVM[];
};
