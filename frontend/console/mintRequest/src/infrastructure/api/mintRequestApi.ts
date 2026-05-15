// frontend/console/mintRequest/src/infrastructure/api/mintRequestApi.ts

import {
  fetchInspectionBatchesHTTP,
  fetchInspectionByProductionIdHTTP,
  fetchInspectionBatchesByProductionIdsHTTP,
  fetchMintListRowsByProductionIdsHTTP,
  fetchMintsByProductionIdsHTTP,
  completeInspectionHTTP,
} from "../repository";

import type { InspectionBatchDTO } from "../../domain/entity/inspections";
import type { MintDTO, MintListRowDTO } from "../dto/mint.dto";

// ===============================
// API 層：Repository 呼び出しラッパ
// ===============================

function normalizeIds(ids: string[]): string[] {
  return (ids ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);
}

/**
 * inspections の一覧を取得する。
 * 内部で /productions → productionIds を作り、/mint/inspections?productionIds=... を叩く。
 */
export async function fetchInspectionBatches(): Promise<InspectionBatchDTO[]> {
  return fetchInspectionBatchesHTTP();
}

/**
 * productionIds を受け取って inspections を取得する。
 */
export async function fetchInspectionBatchesByProductionIds(
  productionIds: string[],
): Promise<InspectionBatchDTO[]> {
  const ids = normalizeIds(productionIds);

  if (ids.length === 0) return [];

  return fetchInspectionBatchesByProductionIdsHTTP(ids);
}

/**
 * mints list row を productionIds でまとめて取得する。
 */
export async function fetchMintsMapByProductionIds(
  productionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = normalizeIds(productionIds);

  if (ids.length === 0) return {};

  const m = await fetchMintListRowsByProductionIdsHTTP(ids);
  return (m ?? {}) as Record<string, MintListRowDTO>;
}

/**
 * MintDTO を productionIds でまとめて取得する。
 */
export async function fetchMintsDTOMapByProductionIds(
  productionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = normalizeIds(productionIds);

  if (ids.length === 0) return {};

  const m = await fetchMintsByProductionIdsHTTP(ids);
  return (m ?? {}) as Record<string, MintDTO>;
}

/**
 * 個別の productionId に紐づく InspectionBatch を取得する。
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

export type { MintDTO, MintListRowDTO };