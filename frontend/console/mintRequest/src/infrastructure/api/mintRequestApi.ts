// frontend/console/mintRequest/src/infrastructure/api/mintRequestApi.ts

import {
  fetchInspectionBatchesHTTP,
  fetchInspectionByProductionIdHTTP,
  fetchInspectionBatchesByProductionIdsHTTP,
} from "../repository";

// ※ 段階移行・型ガードのため、repo 名前空間でも参照しておく
import * as repo from "../repository";

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
  inspectionId?: string | null;
  mintId?: string | null;
  tokenBlueprintId?: string | null;

  tokenName: string;
  createdByName?: string | null;
  mintedAt?: string | null; // 期待: RFC3339 or "yyyy/mm/dd"（どちらでも string として扱う）
  minted?: boolean;
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
 * ✅ inspections の一覧を取得する（新フロー）。
 * 内部で /productions → productionIds を作り、/mint/inspections?productionIds=... を叩く。
 */
export async function fetchInspectionBatches(): Promise<InspectionBatchDTO[]> {
  return fetchInspectionBatchesHTTP();
}

/**
 * ✅ productionIds を受け取って inspections を取得（画面側が productionIds を持っている場合）。
 */
export async function fetchInspectionBatchesByProductionIds(
  productionIds: string[],
): Promise<InspectionBatchDTO[]> {
  const ids = (productionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return [];

  // repository に実装がある場合のみ使う（段階移行に強くする）
  const anyRepo = repo as any;
  if (typeof anyRepo.fetchInspectionBatchesByProductionIdsHTTP === "function") {
    return (await anyRepo.fetchInspectionBatchesByProductionIdsHTTP(ids)) as
      | InspectionBatchDTO[]
      | [];
  }

  // 直接 export しているので基本ここに来ないが、念のため
  return fetchInspectionBatchesByProductionIdsHTTP(ids);
}

/**
 * ✅ mints(list row) を inspectionIds (= productionIds) でまとめて取得する。
 *
 * 推奨: repository の listMintsByInspectionIDsHTTP を優先して呼ぶ
 * （内部で view=list を試し、ダメならフォールバックする実装にしてある）※想定
 */
export async function fetchMintsMapByInspectionIds(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const anyRepo = repo as any;

  // ✅ 推奨: list row 取得（フォールバック付き）
  if (typeof anyRepo.listMintsByInspectionIDsHTTP === "function") {
    const m = await anyRepo.listMintsByInspectionIDsHTTP(ids);
    return (m ?? {}) as Record<string, MintListRowDTO>;
  }
  return {};
}

/**
 * ✅ MintDTO を inspectionIds (= productionIds) でまとめて取得する（肉付け用途）。
 * repository 側に実装が無い場合は {} を返す。
 */
export async function fetchMintsDTOMapByInspectionIds(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const anyRepo = repo as any;

  if (typeof anyRepo.fetchMintsByInspectionIdsHTTP === "function") {
    const m = await anyRepo.fetchMintsByInspectionIdsHTTP(ids);
    return (m ?? {}) as Record<string, MintDTO>;
  }

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
