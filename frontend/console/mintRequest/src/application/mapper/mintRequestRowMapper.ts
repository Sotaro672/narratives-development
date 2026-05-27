// frontend/console/mintRequest/src/application/mapper/mintRequestRowMapper.ts
//
// Query DTO（MintRequestQueryService /mint/requests）専用版
// - productionId = inspectionId の前提で productionId を正とする
// - id / inspectionId / subdoc fallback は扱わない
// - presentation 専用の statusLabel は作らない
// - application/usecase が扱う “行” を返す

import type { InspectionStatus } from "../../domain/entity/inspections";
import type { MintRequestManagementRowDTO } from "../dto/mintRequestManagementRow";

import {
  asNonEmptyString,
  asNumber0,
  asStringOrNull,
} from "../util/primitive";

// ============================================================
// Types (Application row for Management list)
// ============================================================

export type MintRequestRowStatus = "planning" | "requested" | "minted";

export type MintRequestManagementRow = {
  id: string; // productionId

  tokenName: string | null;
  productName: string | null;

  mintQuantity: number;
  productionQuantity: number;

  status: MintRequestRowStatus;
  inspectionStatus: InspectionStatus;

  createdByName: string | null;
  mintedAt: string | null;

  // detail 画面や更新 payload 構築のため保持（表示責務は presentation）
  tokenBlueprintId: string | null;
  requestedBy: string | null;
};

// ============================================================
// Strict helpers
// ============================================================

function requireProductionId(raw: MintRequestManagementRowDTO): string {
  const productionId = asNonEmptyString((raw as any).productionId);

  if (!productionId) {
    throw new Error("MintRequestManagementRowDTO.productionId is required");
  }

  return productionId;
}

function toInspectionStatus(value: unknown): InspectionStatus {
  const status = String(value ?? "").trim();

  if (status === "completed") return "completed";
  if (status === "inspecting") return "inspecting";

  throw new Error(
    `MintRequestManagementRowDTO.inspectionStatus is invalid: ${status}`,
  );
}

// ============================================================
// Status derivation
// ============================================================

export function deriveRowStatus(args: {
  minted: boolean;
  mintedAt: string | null;
  tokenBlueprintId: string | null;
  tokenName: string | null;
  requestedBy: string | null;
}): MintRequestRowStatus {
  if (args.minted || args.mintedAt) return "minted";

  const hasRequestSignal = Boolean(
    args.tokenBlueprintId || args.tokenName || args.requestedBy,
  );

  return hasRequestSignal ? "requested" : "planning";
}

// ============================================================
// Mapper (public)
// ============================================================

/**
 * QueryService の “一覧 DTO” を application 行へ変換して返す。
 * - productionId を正とする
 * - inspectionId / id / mint / inspection subdoc fallback は使わない
 * - presentation 専用の statusLabel は付与しない
 */
export function mapMintRequestManagementRows(
  items: MintRequestManagementRowDTO[],
): MintRequestManagementRow[] {
  return (items ?? []).map((raw) => {
    const id = requireProductionId(raw);

    const tokenName = asStringOrNull((raw as any).tokenName);
    const productName = asStringOrNull((raw as any).productName);

    const mintQuantity = asNumber0((raw as any).mintQuantity);
    const productionQuantity = asNumber0((raw as any).productionQuantity);

    const inspectionStatus = toInspectionStatus(
      (raw as any).inspectionStatus,
    );

    const requestedBy = asStringOrNull((raw as any).requestedBy);

    const createdByName =
      asStringOrNull((raw as any).requestedByName) ??
      asStringOrNull((raw as any).createdByName) ??
      requestedBy;

    const mintedAt = asStringOrNull((raw as any).mintedAt);

    const minted =
      typeof (raw as any).minted === "boolean"
        ? Boolean((raw as any).minted)
        : Boolean(mintedAt);

    const tokenBlueprintId = asStringOrNull(
      (raw as any).tokenBlueprintId,
    );

    const status = deriveRowStatus({
      minted,
      mintedAt,
      tokenBlueprintId,
      tokenName,
      requestedBy,
    });

    return {
      id,

      tokenName,
      productName,

      mintQuantity,
      productionQuantity,

      status,
      inspectionStatus,

      createdByName,
      mintedAt,

      tokenBlueprintId,
      requestedBy,
    } satisfies MintRequestManagementRow;
  });
}