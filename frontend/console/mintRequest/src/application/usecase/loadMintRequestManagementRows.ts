// frontend/console/mintRequest/src/application/usecase/loadMintRequestManagementRows.ts

import type { InspectionStatus } from "../../domain/entity/inspections";
import { fetchMintRequestManagementRowsQueryHTTP } from "../../infrastructure/repository/http/mintRequestManagementQuery";
import type { MintRequestManagementRowDTO } from "../dto/mintRequestManagementRow";

// ============================================================
// Types (list row; kept for current screen expectations)
// ============================================================

export type MintRequestRowStatus = "planning" | "requested" | "minted";

export type ViewRow = {
  id: string; // productionId

  tokenName: string | null;
  productName: string | null;

  mintQuantity: number;
  productionQuantity: number;

  status: MintRequestRowStatus;
  inspectionStatus: InspectionStatus;

  createdByName: string | null;
  mintedAt: string | null;

  tokenBlueprintId: string | null;
  requestedBy: string | null;

  // detail 用（Query が返す前提のフィールドのみ）
  productBlueprintId: string | null;
  scheduledBurnDate: string | null;
  minted: boolean;
};

// ============================================================
// Strict helpers (NO legacy / old-compat fields)
// ============================================================

function asNonEmptyString(v: any): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

function asStringOrNull(v: any): string | null {
  const s = typeof v === "string" ? v.trim() : "";
  return s ? s : null;
}

function asNumber0(v: any): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

function normalizeInspectionStatus(v: any): InspectionStatus {
  const s = String(v ?? "").trim();
  if (s === "completed") return "completed";
  if (s === "inspecting") return "inspecting";
  return "notYet" as any;
}

function deriveRowStatus(args: {
  tokenBlueprintId: string | null;
  tokenName: string | null;
  requestedBy: string | null;
  mintedAt: string | null;
  minted: boolean;
}): MintRequestRowStatus {
  if (args.minted || !!args.mintedAt) return "minted";

  const hasRequestSignal =
    !!asNonEmptyString(args.tokenBlueprintId) ||
    !!asNonEmptyString(args.tokenName) ||
    !!asNonEmptyString(args.requestedBy);

  return hasRequestSignal ? "requested" : "planning";
}

function mapDTOToRow(dto: MintRequestManagementRowDTO): ViewRow {
  // ✅ strict: productionId must exist (旧互換の id / inspectionId は使わない)
  const productionId = asNonEmptyString((dto as any)?.productionId);
  if (!productionId) {
    throw new Error("MintRequestManagementRowDTO.productionId is required");
  }

  const tokenName = asStringOrNull((dto as any)?.tokenName);
  const productName = asStringOrNull((dto as any)?.productName);

  const mintQuantity = asNumber0((dto as any)?.mintQuantity);
  const productionQuantity = asNumber0((dto as any)?.productionQuantity);

  const inspectionStatus = normalizeInspectionStatus((dto as any)?.inspectionStatus);

  const requestedBy = asStringOrNull((dto as any)?.requestedBy);
  const createdByName = asStringOrNull((dto as any)?.createdByName);

  const mintedAt = asStringOrNull((dto as any)?.mintedAt);
  const minted =
    typeof (dto as any)?.minted === "boolean"
      ? Boolean((dto as any)?.minted)
      : Boolean(asNonEmptyString(mintedAt));

  const tokenBlueprintId = asStringOrNull((dto as any)?.tokenBlueprintId);

  const productBlueprintId = asStringOrNull((dto as any)?.productBlueprintId);
  const scheduledBurnDate = asStringOrNull((dto as any)?.scheduledBurnDate);

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

export async function loadMintRequestManagementRows(): Promise<ViewRow[]> {
  const res = await fetchMintRequestManagementRowsQueryHTTP();
  const items = Array.isArray(res?.items) ? res.items : [];
  return items.map(mapDTOToRow);
}
