// frontend/console/mintRequest/src/infrastructure/api/mintRequestApi.ts

import {
  fetchMintRequestsHTTP,
  fetchMintRequestByIdHTTP,
} from "../repository/mintRequestRepositoryHTTP";

// ===============================
// DTO (backend → frontend)
// ===============================

// Go の mintRequest エンティティに対応した素直な DTO
export type MintRequestRowStatus = "planning" | "requested" | "minted";

export type MintRequestDTO = {
  id: string;
  productionId: string;

  status: MintRequestRowStatus | string;

  mintQuantity: number;

  requestedBy?: string | null;
  requestedAt?: string | null; // RFC3339 文字列 or null
  mintedAt?: string | null; // RFC3339 文字列 or null
  scheduledBurnDate?: string | null; // RFC3339 文字列 or null

  tokenBlueprintId?: string | null;

  // 将来的に join で付与するかもしれない拡張フィールド（あれば使う、なければ null）
  productName?: string | null;
  productionQuantity?: number | null;
};

// ===============================
// mintRequestManagement 画面向けの行型
// ===============================

export type MintRequestRow = {
  // 一意キー（従来どおり productionId を採用）
  id: string;

  tokenBlueprintId: string | null;
  productName: string | null;

  mintQuantity: number;
  productionQuantity: number;

  status: MintRequestRowStatus;
  requestedBy: string | null;
  requestedAt: string | null;
  mintedAt: string | null;
};

// DTO → 画面用 MintRequestRow への変換
function mapMintDTOToRow(dto: MintRequestDTO): MintRequestRow {
  const rawStatus = (dto.status || "planning") as MintRequestRowStatus;

  const normalizedStatus: MintRequestRowStatus =
    rawStatus === "requested" || rawStatus === "minted"
      ? rawStatus
      : "planning";

  return {
    // 画面では従来どおり productionId ベースの ID を使う
    id: dto.productionId || dto.id,

    tokenBlueprintId: dto.tokenBlueprintId ?? null,
    productName: dto.productName ?? null,

    mintQuantity: dto.mintQuantity ?? 0,
    // backend 側に productionQuantity が無ければ mintQuantity で代用
    productionQuantity:
      dto.productionQuantity ?? dto.mintQuantity ?? 0,

    status: normalizedStatus,
    requestedBy: dto.requestedBy ?? null,
    requestedAt: dto.requestedAt ?? null,
    mintedAt: dto.mintedAt ?? null,
  };
}

// ===============================
// API 層：Repository 呼び出しラッパ
// ===============================

/**
 * 現在の companyId に紐づく MintRequest 一覧（生 DTO）を取得。
 *   GET /mint-requests
 */
export async function fetchMintRequests(): Promise<MintRequestDTO[]> {
  return fetchMintRequestsHTTP();
}

/**
 * mintRequestManagement 画面向けの行データとして取得するユーティリティ。
 * 画面側では MINT_REQUESTS モックの代わりにこの関数の戻り値を利用する想定。
 */
export async function fetchMintRequestRows(): Promise<MintRequestRow[]> {
  const dtos = await fetchMintRequestsHTTP();
  return dtos.map(mapMintDTOToRow);
}

/**
 * 個別の MintRequest を ID で取得。
 *   GET /mint-requests/{id}
 */
export async function fetchMintRequestById(
  id: string,
): Promise<MintRequestDTO | null> {
  return fetchMintRequestByIdHTTP(id);
}
