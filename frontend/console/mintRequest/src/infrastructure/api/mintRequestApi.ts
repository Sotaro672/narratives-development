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
// ここではその型をそのまま DTO として再利用する。
export type InspectionStatus = DomainInspectionStatus;

// backend/internal/domain/inspection/entity.go に対応
export type InspectionItemDTO = InspectionItem;

// MintUsecase が返す MintInspectionView を、そのまま 1 行分 DTO として扱う
export type InspectionBatchDTO = MintInspectionView;

// ===============================
// mints テーブル（正）DTO
// ===============================
// NOTE: ここでは「mints テーブルのみが正」という前提で、
// requestedBy/requestedAt/mintedAt/tokenBlueprintId は mints 側から解決する。
export type MintDTO = {
  id: string;
  brandId: string;
  tokenBlueprintId: string;
  inspectionId: string; // = productionId（運用）
  products: string[];

  createdAt: string; // ISO string 想定（repo 実装に依存）
  createdBy: string;

  minted: boolean;
  mintedAt?: string | null;

  scheduledBurnDate?: string | null;

  // Firestore 上にあっても、ドメイン未定義なら任意として扱う
  onChainTxSignature?: string | null;
};

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

  // ★ mints テーブル由来
  requestedBy: string | null; // = mints.createdBy
  requestedAt: string | null; // = mints.createdAt
  mintedAt: string | null; // = mints.mintedAt
};

// ===============================
// Mint → MintRequestRow 状態判定
// ===============================

function deriveMintStatusFromMint(mint: MintDTO | null): MintRequestRowStatus {
  if (!mint) return "planning";
  if (mint.minted || !!mint.mintedAt) return "minted";
  return "requested";
}

// ===============================
// Inspection + Mint → MintRequestRow 変換
// ===============================

function mapInspectionToMintRow(
  dto: InspectionBatchDTO,
  mint: MintDTO | null,
): MintRequestRow {
  const requestedAt = mint?.createdAt ?? null;
  const requestedBy = mint?.createdBy ?? null;
  const mintedAt = mint?.mintedAt ?? null;

  return {
    // 画面では従来どおり productionId ベースの ID を使う
    id: dto.productionId,

    // ★ tokenBlueprintId は mints テーブルを正とする
    tokenBlueprintId: mint?.tokenBlueprintId ?? null,

    // ★ バックエンドが MintInspectionView.productName に詰めてくれた値をそのまま利用
    productName: dto.productName ?? null,

    mintQuantity: dto.totalPassed ?? 0,
    // quantity が無ければ inspections.length で代用
    productionQuantity: dto.quantity ?? dto.inspections.length,

    // ★ mints の有無で状態を決める
    status: deriveMintStatusFromMint(mint),

    // ★ 検査ステータス（inspecting / completed）
    inspectionStatus: dto.status,

    // ★ mints 由来
    requestedBy,
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
 * mints を inspectionIds (= productionIds) でまとめて取得する。
 * repository 側が未実装でもコンパイルできるように、存在チェックして安全に noop する。
 *
 * 期待する repository 側の関数（どちらか）:
 * - fetchMintsByInspectionIdsHTTP(ids: string[]): Promise<Record<string, MintDTO>>
 * - fetchMintByInspectionIdHTTP(id: string): Promise<MintDTO | null>
 */
async function fetchMintsMapByInspectionIds(
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
 * mintRequestManagement 画面向けの行データとして取得するユーティリティ。
 * 画面側では MINT_REQUESTS モックの代わりにこの関数の戻り値を利用する想定。
 *
 * - GET /mint/inspections 由来のデータを取得
 * - mints テーブル（正）を inspectionId (= productionId) で引いて突合
 * - MintRequestRow にマッピングする
 */
export async function fetchMintRequestRows(): Promise<MintRequestRow[]> {
  const batches = await fetchInspectionBatchesHTTP();

  const productionIds = batches
    .map((b) => String((b as any).productionId ?? "").trim())
    .filter((s) => !!s);

  const mintMap = await fetchMintsMapByInspectionIds(productionIds);

  return batches.map((b) => {
    const pid = String((b as any).productionId ?? "").trim();
    const mint = pid ? (mintMap[pid] ?? null) : null;
    return mapInspectionToMintRow(b, mint);
  });
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
