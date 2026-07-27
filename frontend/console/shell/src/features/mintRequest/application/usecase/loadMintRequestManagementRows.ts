// frontend/console/shell/src/features/mintRequest/application/usecase/loadMintRequestManagementRows.ts

import type { InspectionStatus } from "../../domain/inspections";

import { fetchMintRequestRowsHTTP } from "../../infrastructure/repository/http/mintRequests";
import { fetchProductionIdsForCurrentCompanyHTTP } from "../../infrastructure/repository/http/productions";

import type { MintRequestManagementRowDTO } from "../dto/mintRequestManagementRow";

import {
  asNonEmptyString,
  asNumber0,
  asStringOrNull,
} from "../util/primitive";

// ============================================================
// Types
// ============================================================

export type MintRequestRowStatus =
  | "planning"
  | "requested"
  | "minted";

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
  inspectionStatus: InspectionStatus;

  /**
   * 既存画面との互換用。
   * requestedByNameと同じ表示名を保持する。
   */
  createdByName: string | null;

  /**
   * ミント申請者の表示名。
   */
  requestedByName: string | null;

  mintedAt: string | null;

  tokenBlueprintId: string | null;
  requestedBy: string | null;

  /**
   * 詳細画面で使用する情報。
   */
  productBlueprintId: string | null;
  scheduledBurnDate: string | null;

  minted: boolean;
};

// ============================================================
// Helpers
// ============================================================

function normalizeInspectionStatus(
  value: unknown,
): InspectionStatus {
  const status = String(value ?? "").trim();

  if (status === "completed") {
    return "completed";
  }

  if (status === "inspecting") {
    return "inspecting";
  }

  /**
   * InspectionStatusの型定義は
   * "inspecting" | "completed" だが、
   * 一覧APIでは検品レコードが存在しない場合に
   * "notYet" が返る。
   *
   * 現行画面との互換を維持するため、
   * ここではInspectionStatusとして扱う。
   */
  return "notYet" as InspectionStatus;
}

function deriveRowStatus(args: {
  tokenBlueprintId: string | null;
  tokenName: string | null;
  requestedBy: string | null;
  mintedAt: string | null;
  minted: boolean;
}): MintRequestRowStatus {
  if (args.minted || Boolean(args.mintedAt)) {
    return "minted";
  }

  const hasRequestSignal =
    Boolean(
      asNonEmptyString(args.tokenBlueprintId),
    ) ||
    Boolean(
      asNonEmptyString(args.tokenName),
    ) ||
    Boolean(
      asNonEmptyString(args.requestedBy),
    );

  return hasRequestSignal
    ? "requested"
    : "planning";
}

function mapDTOToRow(
  dto: MintRequestManagementRowDTO,
): ViewRow {
  const raw = dto as Record<string, unknown>;

  /**
   * productionIdを正とする。
   * id / inspectionIdによる旧形式のfallbackは行わない。
   */
  const productionId = asNonEmptyString(
    raw.productionId,
  );

  if (!productionId) {
    throw new Error(
      "MintRequestManagementRowDTO.productionId is required",
    );
  }

  const tokenName = asStringOrNull(
    raw.tokenName,
  );

  const productName = asStringOrNull(
    raw.productName,
  );

  const mintQuantity = asNumber0(
    raw.mintQuantity,
  );

  const productionQuantity = asNumber0(
    raw.productionQuantity,
  );

  const inspectionStatus =
    normalizeInspectionStatus(
      raw.inspectionStatus,
    );

  const requestedBy = asStringOrNull(
    raw.requestedBy,
  );

  /**
   * requestedByNameを正とする。
   *
   * 現行Backendとの互換のため、値がない場合のみ
   * createdByName、requestedByの順で補完する。
   */
  const requestedByName =
    asStringOrNull(
      raw.requestedByName,
    ) ??
    asStringOrNull(
      raw.createdByName,
    ) ??
    requestedBy;

  /**
   * 既存画面との互換用フィールド。
   */
  const createdByName = requestedByName;

  const mintedAt = asStringOrNull(
    raw.mintedAt,
  );

  const minted =
    typeof raw.minted === "boolean"
      ? raw.minted
      : Boolean(
          asNonEmptyString(mintedAt),
        );

  const tokenBlueprintId =
    asStringOrNull(
      raw.tokenBlueprintId,
    );

  const productBlueprintId =
    asStringOrNull(
      raw.productBlueprintId,
    );

  const scheduledBurnDate =
    asStringOrNull(
      raw.scheduledBurnDate,
    );

  const status = deriveRowStatus({
    tokenBlueprintId,
    tokenName,
    requestedBy,
    mintedAt,
    minted,
  });

  return {
    id: productionId,

    tokenName,
    productName,

    mintQuantity,
    productionQuantity,

    status,
    inspectionStatus,

    createdByName,
    requestedByName,
    mintedAt,

    tokenBlueprintId,
    requestedBy,

    productBlueprintId,
    scheduledBurnDate,

    minted,
  };
}

// ============================================================
// Usecase
// ============================================================

/**
 * 現在の会社に属するミント申請一覧を取得する。
 *
 * 1. /productionsからproductionIdsを取得する
 * 2. 統合済みのfetchMintRequestRowsHTTPを使って
 *    GET /mint/requestsを1回だけ実行する
 * 3. Backend DTOを一覧画面用ViewRowへ変換する
 */
export async function loadMintRequestManagementRows(): Promise<
  ViewRow[]
> {
  const productionIds =
    await fetchProductionIdsForCurrentCompanyHTTP();

  if (productionIds.length === 0) {
    return [];
  }

  const rows = await fetchMintRequestRowsHTTP(
    productionIds,
    "list",
  );

  return rows.map(mapDTOToRow);
}