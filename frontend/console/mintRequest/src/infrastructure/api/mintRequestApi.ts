// frontend/console/mintRequest/src/infrastructure/api/mintRequestApi.ts

import {
  fetchInspectionBatchesHTTP,
  fetchInspectionByProductionIdHTTP,
  fetchInspectionBatchesByProductionIdsHTTP,
  fetchMintListRowsByInspectionIdsHTTP,
  fetchMintsByInspectionIdsHTTP,
  completeInspectionHTTP,
} from "../repository";

import type { InspectionBatchDTO } from "../../domain/entity/inspections";

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

  return fetchInspectionBatchesByProductionIdsHTTP(ids);
}

/**
 * ✅ mints(list row) を inspectionIds (= productionIds) でまとめて取得する。
 */
export async function fetchMintsMapByInspectionIds(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const m = await fetchMintListRowsByInspectionIdsHTTP(ids);
  return (m ?? {}) as Record<string, MintListRowDTO>;
}

/**
 * ✅ MintDTO を inspectionIds (= productionIds) でまとめて取得する（肉付け用途）。
 */
export async function fetchMintsDTOMapByInspectionIds(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const m = await fetchMintsByInspectionIdsHTTP(ids);
  return (m ?? {}) as Record<string, MintDTO>;
}

/**
 * 個別の productionId に紐づく InspectionBatch を取得。
 * 詳細画面などでの利用を想定。
 */
export async function fetchInspectionByProductionId(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const id = String(productionId ?? "").trim();
  if (!id) return null;

  return fetchInspectionByProductionIdHTTP(id);
}

/**
 * productionId に紐づく inspection を検品完了にする。
 *
 * ネガティブ制では、complete 時に notYet の productId が passed として確定される。
 */
export async function completeInspectionByProductionId(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const id = String(productionId ?? "").trim();

  if (!id) {
    throw new Error("productionId is required");
  }

  return completeInspectionHTTP(id);
}