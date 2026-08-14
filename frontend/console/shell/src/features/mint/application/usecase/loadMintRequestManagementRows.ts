// frontend/console/shell/src/features/mint/application/usecase/loadMintRequestManagementRows.ts

import type { InspectionStatus } from "../../../../shared/types/inspections";
import type { MintStatus } from "../../../../shared/types/mints";

import { fetchMintRequestRowsHTTP } from "../../infrastructure/repository/http/mintRequests";
import type { MintRequestManagementRowDTO } from "../../infrastructure/dto/mintRequestManagementRow";

// ============================================================
// Types
// ============================================================

export type MintRequestRowStatus =
  | "planning"
  | "requested"
  | "minting"
  | "minted";

export type MintRequestInspectionStatus =
  | InspectionStatus
  | "notYet";

export type ViewRow = {
  productionId: string;
  tokenBlueprintId: string | null;
  tokenName: string | null;
  productName: string | null;
  mintQuantity: number;
  productionQuantity: number;
  mintStatus: MintRequestManagementRowDTO["mintStatus"];
  status: MintRequestRowStatus;
  inspectionStatus: MintRequestInspectionStatus;
  createdBy: string | null;
  createdByName: string | null;
  requestedBy: string | null;
  requestedByName: string | null;
  mintedAt: string | null;
};

// ============================================================
// Helpers
// ============================================================

/**
 * Backendの親Mint状態から一覧表示用の状態を算出する。
 *
 * BackendのmintStatusを唯一の正とし、
 * tokenBlueprintId / tokenName / requestedBy / mintedAtなどによる再判定は行わない。
 */
function deriveRowStatus(
  mintStatus: MintStatus | string | null | undefined,
): MintRequestRowStatus {
  switch (mintStatus) {
    case "MINTED":
      return "minted";

    case "QUEUED":
    case "MINTING":
    case "PARTIALLY_MINTED":
      return "minting";

    case "CREATED":
    case "FAILED_RETRYABLE":
    case "FAILED_FATAL":
      return "requested";

    default:
      return "planning";
  }
}

/**
 * Backend BFF DTOを一覧表示用ViewRowへ変換する。
 *
 * productionId / mintStatus / mintQuantity / productionQuantityはBackend responseを正としてそのまま保持する。
 * UI表示用のstatusのみFrontendで導出する。
 */
function mapDTOToRow(
  dto: MintRequestManagementRowDTO,
): ViewRow {
  return {
    productionId: dto.productionId,
    tokenBlueprintId: dto.tokenBlueprintId ?? null,
    tokenName: dto.tokenName ?? null,
    productName: dto.productName ?? null,
    mintQuantity: dto.mintQuantity,
    productionQuantity: dto.productionQuantity,
    mintStatus: dto.mintStatus,
    status: deriveRowStatus(dto.mintStatus),
    inspectionStatus: dto.inspectionStatus as MintRequestInspectionStatus,
    createdBy: dto.createdBy ?? null,
    createdByName: dto.createdByName ?? null,
    requestedBy: dto.requestedBy ?? null,
    requestedByName: dto.requestedByName ?? null,
    mintedAt: dto.mintedAt ?? null,
  };
}

// ============================================================
// Usecase
// ============================================================

/**
 * 現在の会社に属するミント申請一覧を取得する。
 *
 * BackendのGET /mint/requestsを一覧データの唯一の正とする。
 * productionIdsをFrontendで事前取得せず、空配列を渡した場合はBackend側で現在companyのproductionを解決する。
 */
export async function loadMintRequestManagementRows(): Promise<ViewRow[]> {
  const rows = await fetchMintRequestRowsHTTP([]);

  return rows.map(mapDTOToRow);
}