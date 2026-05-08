// frontend/console/mintRequest/src/application/mapper/mintRequestRowMapper.ts
//
// Query DTO（MintRequestQueryService /mint/requests）専用版
// - legacy/join（inspections + mints + productions）を廃止
// - presentation 専用の statusLabel は作らない
// - application/usecase が扱う “行” を返す

import type { InspectionStatus } from "../../domain/entity/inspections";
import type { MintRequestManagementRowDTO } from "../dto/mintRequestManagementRow";

import {
  asNonEmptyString,
  asStringOrNull,
  asNumber0,
} from "../util/primitive";

// ============================================================
// Types (Application row for Management list)
// ============================================================

export type MintRequestRowStatus = "planning" | "requested" | "minted";

export type MintRequestManagementRow = {
  id: string; // productionId (= inspectionId 扱い)

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
// Normalizers
// ============================================================

export function normalizeInspectionStatus(v: any): InspectionStatus {
  const s = String(v ?? "").trim();
  if (s === "completed") return "completed";
  if (s === "inspecting") return "inspecting";
  // domain 側に notYet が存在する前提
  return "notYet" as any;
}

export function normalizeRowId(raw: MintRequestManagementRowDTO): string {
  return (
    asNonEmptyString(raw.productionId) ||
    asNonEmptyString(raw.inspectionId) ||
    asNonEmptyString(raw.id)
  );
}

export function pickMintedAt(raw: MintRequestManagementRowDTO): string | null {
  return asStringOrNull(raw.mintedAt) ?? asStringOrNull(raw.mint?.mintedAt);
}

export function pickTokenName(raw: MintRequestManagementRowDTO): string | null {
  return asStringOrNull(raw.tokenName) ?? asStringOrNull(raw.mint?.tokenName);
}

export function pickProductName(raw: MintRequestManagementRowDTO): string | null {
  return (
    asStringOrNull(raw.productName) ?? asStringOrNull(raw.inspection?.productName)
  );
}

export function pickMintQuantity(raw: MintRequestManagementRowDTO): number {
  // ✅ backend は mintQuantity / productionQuantity を返すので最優先
  return asNumber0(raw.mintQuantity ?? raw.totalPassed ?? raw.inspection?.totalPassed ?? 0);
}

export function pickProductionQuantity(raw: MintRequestManagementRowDTO): number {
  return asNumber0(
    raw.productionQuantity ?? raw.quantity ?? raw.inspection?.quantity ?? 0,
  );
}

export function pickTokenBlueprintId(raw: MintRequestManagementRowDTO): string | null {
  return (
    asStringOrNull(raw.tokenBlueprintId) ??
    asStringOrNull(raw.mint?.tokenBlueprintId) ??
    asStringOrNull(raw.mint?.tokenBlueprintID) ??
    asStringOrNull(raw.mint?.tokenBlueprint)
  );
}

export function pickRequestedBy(raw: MintRequestManagementRowDTO): string | null {
  // requestedBy = mint.createdBy（想定）
  return asStringOrNull(raw.requestedBy) ?? asStringOrNull(raw.mint?.createdBy);
}

export function pickRequesterName(raw: MintRequestManagementRowDTO): string | null {
  // 表示は createdByName/requestedByName を優先するが、application は値だけ返す
  return (
    asStringOrNull(raw.requestedByName) ??
    asStringOrNull(raw.createdByName) ??
    pickRequestedBy(raw)
  );
}

// ============================================================
// Status derivation
// ============================================================

export function deriveRowStatus(
  mintedAt: string | null,
  tokenBlueprintId: string | null,
  tokenName: string | null,
  requestedBy: string | null,
): MintRequestRowStatus {
  if (mintedAt) return "minted";

  // “申請が存在する” シグナルがあれば requested
  const hasRequestSignal = Boolean(
    tokenBlueprintId || tokenName || requestedBy,
  );

  return hasRequestSignal ? "requested" : "planning";
}

// ============================================================
// Mapper (public)
// ============================================================

/**
 * QueryService の “一覧 DTO” を application 行へ正規化して返す。
 * - ここでは presentation 専用の statusLabel は付与しない
 */
export function mapMintRequestManagementRows(
  items: MintRequestManagementRowDTO[],
): MintRequestManagementRow[] {
  return (items ?? [])
    .map((raw) => {
      const id = normalizeRowId(raw);
      if (!id) return null;

      const inspectionStatus = normalizeInspectionStatus(
        raw.inspectionStatus ?? raw.inspection?.status,
      );

      const mintedAt = pickMintedAt(raw);
      const tokenName = pickTokenName(raw);
      const productName = pickProductName(raw);

      const mintQuantity = pickMintQuantity(raw);
      const productionQuantity = pickProductionQuantity(raw);

      const tokenBlueprintId = pickTokenBlueprintId(raw);
      const requestedBy = pickRequestedBy(raw);
      const createdByName = pickRequesterName(raw);

      const status = deriveRowStatus(
        mintedAt,
        tokenBlueprintId,
        tokenName,
        requestedBy,
      );

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
    })
    .filter((v): v is MintRequestManagementRow => v !== null);
}
