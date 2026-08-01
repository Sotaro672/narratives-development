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

import {
  fetchProductionIdsForCurrentCompanyHTTP,
} from "../../infrastructure/repository/http/productions";

import type {
  MintRequestManagementRowDTO,
} from "../../infrastructure/dto/mintRequestManagementRow";

import {
  asNonEmptyString,
  asNumber0,
  asStringOrNull,
} from "../../../../shared/util/primitive";

// ============================================================
// Types
// ============================================================

export type MintRequestRowStatus =
  | "planning"
  | "requested"
  | "minting"
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
  const status =
    String(value ?? "").trim();

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

/**
 * Backendから返される親Mintの状態を正規化する。
 *
 * Backendでは大文字のMintStatusを正とする。
 * 不明な値や空文字はnullとして扱う。
 */
function normalizeMintStatus(
  value: unknown,
): MintStatus | null {
  const status =
    String(value ?? "")
      .trim()
      .toUpperCase();

  switch (status) {
    case "CREATED":
    case "QUEUED":
    case "MINTING":
    case "PARTIALLY_MINTED":
    case "MINTED":
    case "FAILED_RETRYABLE":
    case "FAILED_FATAL":
      return status;

    default:
      return null;
  }
}

/**
 * Backendの親Mint状態と、一覧DTOの各項目から
 * 一覧表示用の状態を算出する。
 */
function deriveRowStatus(args: {
  tokenBlueprintId: string | null;
  tokenName: string | null;
  requestedBy: string | null;
  mintedAt: string | null;
  minted: boolean;
  mintStatus: MintStatus | null;
}): MintRequestRowStatus {
  /**
   * Mint完了を最優先する。
   */
  if (
    args.mintStatus === "MINTED" ||
    args.minted ||
    Boolean(
      asNonEmptyString(
        args.mintedAt,
      ),
    )
  ) {
    return "minted";
  }

  /**
   * 非同期Mint処理の受付後から完了前までを
   * 一覧上では「ミント中」として扱う。
   */
  if (
    args.mintStatus === "QUEUED" ||
    args.mintStatus === "MINTING" ||
    args.mintStatus ===
      "PARTIALLY_MINTED"
  ) {
    return "minting";
  }

  /**
   * CREATEDまたは失敗状態の場合でも、
   * Mint申請を示す情報が存在すれば
   * 申請済みとして扱う。
   */
  const hasRequestSignal =
    Boolean(
      asNonEmptyString(
        args.tokenBlueprintId,
      ),
    ) ||
    Boolean(
      asNonEmptyString(
        args.tokenName,
      ),
    ) ||
    Boolean(
      asNonEmptyString(
        args.requestedBy,
      ),
    );

  return hasRequestSignal
    ? "requested"
    : "planning";
}

function mapDTOToRow(
  dto: MintRequestManagementRowDTO,
): ViewRow {
  const raw =
    dto as Record<string, unknown>;

  /**
   * Backendが返すネストされたMint概要。
   *
   * mint.statusを優先し、
   * トップレベルのstatusは互換用として使用する。
   */
  const mintRaw =
    raw.mint &&
    typeof raw.mint === "object" &&
    !Array.isArray(raw.mint)
      ? (
          raw.mint as Record<
            string,
            unknown
          >
        )
      : null;

  const mintStatus =
    normalizeMintStatus(
      mintRaw?.status ??
        raw.status,
    );

  /**
   * productionIdを正とする。
   * id / inspectionIdによる旧形式のfallbackは行わない。
   */
  const productionId =
    asNonEmptyString(
      raw.productionId,
    );

  if (!productionId) {
    throw new Error(
      "MintRequestManagementRowDTO.productionId is required",
    );
  }

  const tokenName =
    asStringOrNull(
      raw.tokenName,
    );

  const productName =
    asStringOrNull(
      raw.productName,
    );

  const mintQuantity =
    asNumber0(
      raw.mintQuantity,
    );

  const productionQuantity =
    asNumber0(
      raw.productionQuantity,
    );

  const inspectionStatus =
    normalizeInspectionStatus(
      raw.inspectionStatus,
    );

  /**
   * mintsドキュメントを作成したメンバー。
   *
   * requestedBy系からのfallbackは行わない。
   */
  const createdBy =
    asStringOrNull(
      raw.createdBy,
    );

  /**
   * createdByNameがない場合は、
   * 対応するcreatedByのみを表示値として使用する。
   */
  const createdByName =
    asStringOrNull(
      raw.createdByName,
    ) ??
    createdBy;

  /**
   * Mint申請ボタンを押したメンバー。
   *
   * createdBy系からのfallbackは行わない。
   */
  const requestedBy =
    asStringOrNull(
      raw.requestedBy,
    );

  /**
   * requestedByNameがない場合は、
   * 対応するrequestedByのみを表示値として使用する。
   */
  const requestedByName =
    asStringOrNull(
      raw.requestedByName,
    ) ??
    requestedBy;

  const mintedAt =
    asStringOrNull(
      raw.mintedAt,
    ) ??
    asStringOrNull(
      mintRaw?.mintedAt,
    );

  /**
   * mintedフラグ、MintStatus、mintedAtの
   * いずれかでMint完了を判定する。
   */
  const minted =
    raw.minted === true ||
    mintStatus === "MINTED" ||
    Boolean(
      asNonEmptyString(
        mintedAt,
      ),
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

  const status =
    deriveRowStatus({
      tokenBlueprintId,
      tokenName,
      requestedBy,
      mintedAt,
      minted,
      mintStatus,
    });

  return {
    id: productionId,

    tokenName,
    productName,

    mintQuantity,
    productionQuantity,

    status,
    inspectionStatus,

    createdBy,
    createdByName,

    requestedBy,
    requestedByName,

    mintedAt,

    tokenBlueprintId,

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

  const rows =
    await fetchMintRequestRowsHTTP(
      productionIds,
      "list",
    );

  return rows.map(
    mapDTOToRow,
  );
}