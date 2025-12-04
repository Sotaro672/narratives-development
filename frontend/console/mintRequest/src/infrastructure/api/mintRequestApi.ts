// frontend/console/mintRequest/src/infrastructure/api/mintRequestApi.ts

import {
  fetchInspectionBatchesHTTP,
  fetchInspectionByProductionIdHTTP,
} from "../repository/mintRequestRepositoryHTTP";
import type {
  InspectionBatch,
  InspectionItem,
  InspectionStatus as DomainInspectionStatus,
  MintInspectionView,
} from "../../domain/entity/inspections";

// ===============================
// DTO (backend → frontend)
// ===============================

// frontend/console/mintRequest/src/domain/entity/inspections.ts を正とする。
// ここではその型をそのまま DTO として再利用する。

export type InspectionStatus = DomainInspectionStatus;

// backend/internal/domain/inspection/entity.go に対応
export type InspectionItemDTO = InspectionItem;

// MintUsecase が返す MintInspectionView を、そのまま 1 行分 DTO として扱う
export type InspectionBatchDTO = MintInspectionView;

// ===============================
// mintRequestManagement 画面向けの行型
// ===============================

export type MintRequestRowStatus = "planning" | "requested" | "minted";

export type MintRequestRow = {
  // 一意キー（従来どおり productionId を採用）
  id: string;

  tokenBlueprintId: string | null;

  // productBlueprintId から解決されたプロダクト名（バックエンド側で join 済み想定）
  productName: string | null;

  mintQuantity: number;
  productionQuantity: number;

  // Mint リクエストの状態（planning / requested / minted）
  status: MintRequestRowStatus;

  // 検査ステータス（inspecting / completed）
  inspectionStatus: InspectionStatus;

  requestedBy: string | null;
  requestedAt: string | null;
  mintedAt: string | null;
};

// ===============================
// Inspection → MintRequestRow 変換
// ===============================

// InspectionStatus / requestedAt / mintedAt から MintRequestStatus を推定
function deriveMintStatus(
  status: InspectionStatus,
  requestedAt: string | null,
  mintedAt: string | null,
): MintRequestRowStatus {
  // Mint 完了日時があれば minted 優先
  if (mintedAt) {
    return "minted";
  }
  // リクエスト日時があれば requested
  if (requestedAt) {
    return "requested";
  }
  // それ以外は planning 扱い
  return "planning";
}

// InspectionBatchDTO(MintInspectionView) → 画面用 MintRequestRow への変換
function mapInspectionToMintRow(dto: InspectionBatchDTO): MintRequestRow {
  const requestedAt = dto.requestedAt ?? null;
  const mintedAt = dto.mintedAt ?? null;

  return {
    // 画面では従来どおり productionId ベースの ID を使う
    id: dto.productionId,

    tokenBlueprintId: dto.tokenBlueprintId ?? null,

    // ★ バックエンドが MintInspectionView.productName に詰めてくれた値をそのまま利用
    productName: dto.productName ?? null,

    mintQuantity: dto.totalPassed ?? 0,
    // quantity が無ければ inspections.length で代用
    productionQuantity: dto.quantity ?? dto.inspections.length,

    // Mint の状態を推定
    status: deriveMintStatus(dto.status, requestedAt, mintedAt),

    // ★ 検査ステータスをそのまま保持（inspecting / completed）
    inspectionStatus: dto.status,

    requestedBy: dto.requestedBy ?? null,
    requestedAt,
    mintedAt,
  };
}

// ===============================
// API 層：Repository 呼び出しラッパ
// ===============================

/**
 * inspections の一覧をそのまま取得する（汎用用途向け）。
 */
export async function fetchInspectionBatches(): Promise<InspectionBatchDTO[]> {
  return fetchInspectionBatchesHTTP();
}

/**
 * mintRequestManagement 画面向けの行データとして取得するユーティリティ。
 * 画面側では MINT_REQUESTS モックの代わりにこの関数の戻り値を利用する想定。
 *   GET /mint/inspections 由来のデータを MintRequestRow にマッピングする。
 */
export async function fetchMintRequestRows(): Promise<MintRequestRow[]> {
  const batches = await fetchInspectionBatchesHTTP();
  return batches.map(mapInspectionToMintRow);
}

/**
 * 個別の productionId に紐づく InspectionBatch を取得。
 * 詳細画面などでの利用を想定。
 */
export async function fetchInspectionByProductionId(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  return fetchInspectionByProductionIdHTTP(productionId);
}
