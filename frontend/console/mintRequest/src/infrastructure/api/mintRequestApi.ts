// frontend/console/mintRequest/src/infrastructure/api/mintRequestApi.ts

import {
  fetchInspectionBatchesHTTP,
  fetchInspectionByProductionIdHTTP,
} from "../repository/mintRequestRepositoryHTTP";
import * as repo from "../repository/mintRequestRepositoryHTTP";

import type {
  InspectionItem,
  InspectionStatus as DomainInspectionStatus,
  MintInspectionView,
} from "../../domain/entity/inspections";

// ===============================
// DTO (backend → frontend)
// ===============================

// frontend/console/mintRequest/src/domain/entity/inspections.ts を正とする。
export type InspectionStatus = DomainInspectionStatus;

// backend/internal/domain/inspection/entity.go に対応
export type InspectionItemDTO = InspectionItem;

// MintUsecase が返す MintInspectionView を、そのまま 1 行分 DTO として扱う
export type InspectionBatchDTO = MintInspectionView;

// ===============================
// mints テーブル（LIST 用の最小 DTO）
// ===============================
// backend/internal/application/mint/dto/list.go の MintListRowDTO を正とする。
// inspectionId (= productionId) をキーにした map として返ってくる想定。
export type MintListRowDTO = {
  tokenName: string;
  createdByName?: string | null;
  mintedAt?: string | null; // 期待: "yyyy/mm/dd"（バックエンド整形）
};

// ===============================
// mints テーブル（DETAIL 用の DTO）
// ===============================
// NOTE:
// - detailed fields は detail API が担う想定だが、既存互換のため残しておく
export type MintDTO = {
  id: string;
  brandId: string;
  tokenBlueprintId: string;
  inspectionId: string; // = productionId（運用）
  products: string[];

  createdAt: string; // ISO string 想定（repo 実装に依存）
  createdBy: string;
  createdByName?: string | null;

  minted: boolean;
  mintedAt?: string | null;

  scheduledBurnDate?: string | null;

  onChainTxSignature?: string | null;
};

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
 * mints(list row) を inspectionIds (= productionIds) でまとめて取得する。
 *
 * 期待する repository 側の関数:
 * - fetchMintsByInspectionIdsHTTP(ids: string[]): Promise<Record<string, MintListRowDTO>>
 *
 * （単発取得しかない場合は MintDTO を返す可能性があるため、ここでは扱わない）
 */
export async function fetchMintsMapByInspectionIds(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const anyRepo = repo as any;

  if (typeof anyRepo.fetchMintsByInspectionIdsHTTP === "function") {
    const m = await anyRepo.fetchMintsByInspectionIdsHTTP(ids);
    return (m ?? {}) as Record<string, MintListRowDTO>;
  }

  // 未実装なら空
  return {};
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
