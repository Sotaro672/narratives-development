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
// mints テーブル（正）DTO
// ===============================
// NOTE:
// - requestedBy/requestedAt は使わない（mints.createdBy / createdAt が正）
// - createdByName は UI 表示用途のため追加（無い場合は createdBy にフォールバック）
export type MintDTO = {
  id: string;
  brandId: string;
  tokenBlueprintId: string;
  inspectionId: string; // = productionId（運用）
  products: string[];

  createdAt: string; // ISO string 想定（repo 実装に依存）
  createdBy: string;
  createdByName?: string | null; // ★ 追加（mints.createdByName）

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
 * mints を inspectionIds (= productionIds) でまとめて取得する。
 * repository 側が未実装でもコンパイルできるように、存在チェックして安全に noop する。
 *
 * 期待する repository 側の関数（どちらか）:
 * - fetchMintsByInspectionIdsHTTP(ids: string[]): Promise<Record<string, MintDTO>>
 * - fetchMintByInspectionIdHTTP(id: string): Promise<MintDTO | null>
 */
export async function fetchMintsMapByInspectionIds(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const anyRepo = repo as any;

  // 1) 一括取得があるならそれを優先
  if (typeof anyRepo.fetchMintsByInspectionIdsHTTP === "function") {
    const m = await anyRepo.fetchMintsByInspectionIdsHTTP(ids);
    return (m ?? {}) as Record<string, MintDTO>;
  }

  // 2) 単発取得しかない場合は並列で引く
  if (typeof anyRepo.fetchMintByInspectionIdHTTP === "function") {
    const pairs = await Promise.all(
      ids.map(async (id) => {
        try {
          const mint = (await anyRepo.fetchMintByInspectionIdHTTP(id)) as
            | MintDTO
            | null;
          return [id, mint] as const;
        } catch {
          return [id, null] as const;
        }
      }),
    );

    const out: Record<string, MintDTO> = {};
    for (const [id, mint] of pairs) {
      if (mint) out[id] = mint;
    }
    return out;
  }

  // 3) 未実装なら空
  return {};
}

/**
 * mintRequestManagement 画面では MintDTO を正として扱う。
 * - inspections を取得して productionIds を作り
 * - mints をまとめて取得し
 * - MintDTO を配列で返す（Hook 側で inspections と突合して表示行を作る）
 */
export async function fetchMintRequestRows(): Promise<MintDTO[]> {
  const batches = await fetchInspectionBatchesHTTP();

  const productionIds = batches
    .map((b) => String((b as any).productionId ?? "").trim())
    .filter((s) => !!s);

  const mintMap = await fetchMintsMapByInspectionIds(productionIds);

  // ★ 画面側は「inspections に存在する productionId を母集団」にするので、
  //   batches 順で MintDTO を返す（mint が無い場合は返さない）
  const out: MintDTO[] = [];
  for (const pid of productionIds) {
    const m = mintMap[pid];
    if (m) out.push(m);
  }
  return out;
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
