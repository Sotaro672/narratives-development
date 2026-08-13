// frontend/console/shell/src/features/mintRequest/application/usecase/loadMintRequestManagementRows.ts

import type {
  InspectionStatus,
} from "../../../../shared/types/inspections";

import type {
  MintStatus,
} from "../../../../shared/types/mints";

import {
  fetchMintRequestRowsHTTP,
} from "../../infrastructure/repository/http/mintRequests";

import type {
  MintRequestManagementRowDTO,
} from "../../infrastructure/dto/mintRequestManagementRow";

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
  /**
   * productionId
   */
  id: string;

  tokenName: string | null;
  productName: string | null;

  mintQuantity: number;
  productionQuantity: number;

  status: MintRequestRowStatus;
  inspectionStatus: MintRequestInspectionStatus;

  /**
   * mintsドキュメントを作成したmemberId。
   */
  createdBy: string | null;

  /**
   * mintsドキュメントを作成したメンバーの表示名。
   */
  createdByName: string | null;

  /**
   * Mint申請ボタンを押したmemberId。
   */
  requestedBy: string | null;

  /**
   * Mint申請ボタンを押したメンバーの表示名。
   */
  requestedByName: string | null;

  mintedAt: string | null;

  tokenBlueprintId: string | null;

  minted: boolean;
};

// ============================================================
// Helpers
// ============================================================

/**
 * Backendの親Mint状態から
 * 一覧表示用の状態を算出する。
 *
 * BackendのmintStatusを唯一の正とし、
 * tokenBlueprintId / tokenName / requestedBy /
 * mintedAtなどによる再判定は行わない。
 */
function deriveRowStatus(
  mintStatus:
    | MintStatus
    | null
    | undefined,
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
 * Backend responseのfield名・型・値を正とし、
 * 旧response形式のfallbackや値の正規化は行わない。
 */
function mapDTOToRow(
  dto: MintRequestManagementRowDTO,
): ViewRow {
  if (!dto.productionId) {
    throw new Error(
      "MintRequestManagementRowDTO.productionId is required",
    );
  }

  const mintStatus =
    dto.mintStatus as
      | MintStatus
      | null
      | undefined;

  return {
    id:
      dto.productionId,

    tokenName:
      dto.tokenName ?? null,

    productName:
      dto.productName ?? null,

    mintQuantity:
      dto.mintQuantity!,

    productionQuantity:
      dto.productionQuantity!,

    status:
      deriveRowStatus(
        mintStatus,
      ),

    inspectionStatus:
      dto.inspectionStatus as
        MintRequestInspectionStatus,

    createdBy:
      dto.createdBy ?? null,

    createdByName:
      dto.createdByName ?? null,

    requestedBy:
      dto.requestedBy ?? null,

    requestedByName:
      dto.requestedByName ?? null,

    mintedAt:
      dto.mintedAt ?? null,

    tokenBlueprintId:
      dto.tokenBlueprintId ?? null,

    minted:
      mintStatus === "MINTED",
  };
}

// ============================================================
// Usecase
// ============================================================

/**
 * 現在の会社に属するミント申請一覧を取得する。
 *
 * BackendのGET /mint/requests?view=listを
 * 一覧データの唯一の正とする。
 *
 * productionIdsをFrontendで事前取得せず、
 * Backend側で現在companyのproductionを解決する。
 */
export async function loadMintRequestManagementRows(): Promise<
  ViewRow[]
> {
  const rows =
    await fetchMintRequestRowsHTTP(
      [],
      "list",
    );

  return rows.map(
    mapDTOToRow,
  );
}