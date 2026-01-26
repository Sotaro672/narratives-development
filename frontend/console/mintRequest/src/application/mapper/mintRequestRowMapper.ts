// frontend/console/mintRequest/src/application/mapper/mintRequestRowMapper.ts

import type { InspectionStatus } from "../../domain/entity/inspections";
import type {
  InspectionBatchDTO,
  MintDTO,
  MintListRowDTO,
} from "../../infrastructure/api/mintRequestApi";

// ============================================================
// Types
// ============================================================

export type MintRequestRowStatus = "planning" | "requested" | "minted";

export type ViewRow = {
  id: string; // = productionId (= inspectionId 扱い)

  tokenName: string | null;
  productName: string | null;

  mintQuantity: number; // = inspection.totalPassed
  productionQuantity: number; // = inspection.quantity (fallback: production.totalQuantity)

  status: MintRequestRowStatus;
  inspectionStatus: InspectionStatus;

  createdByName: string | null;
  mintedAt: string | null;

  tokenBlueprintId: string | null;
  requestedBy: string | null;

  productBlueprintId: string | null;
  scheduledBurnDate: string | null;
  minted: boolean;

  statusLabel: string;
};

export type ProductionIndex = {
  productionIds: string[];
  productNameById: Record<string, string | null>;
  totalQuantityById: Record<string, number>;
  productBlueprintIdById: Record<string, string | null>;
};

// ============================================================
// small helpers
// ============================================================

export function asTrimmedString(v: any): string {
  return typeof v === "string" ? v.trim() : String(v ?? "").trim();
}

export function asMaybeString(v: any): string | null {
  const s = asTrimmedString(v);
  return s ? s : null;
}

export function normalizeProductionId(v: any): string {
  return String(v?.productionId ?? v?.inspectionId ?? v?.id ?? "").trim();
}

export function inspectionStatusLabel(
  s: InspectionStatus | null | undefined,
): string {
  switch (s) {
    case "inspecting":
      return "検査中";
    case "completed":
      return "検査完了";
    default:
      return "未検査";
  }
}

export function deriveMintStatusFromMintDTO(
  mint: MintDTO | null,
): MintRequestRowStatus {
  if (!mint) return "planning";
  if ((mint as any)?.mintedAt) return "minted";
  if ((mint as any)?.minted === true) return "minted";
  return "requested";
}

export function pickTokenBlueprintId(mintDTO: any, mintList: any): string | null {
  const v =
    mintDTO?.tokenBlueprintId ??
    mintDTO?.TokenBlueprintID ??
    mintList?.tokenBlueprintId ??
    mintList?.TokenBlueprintID ??
    null;

  const s = typeof v === "string" ? v.trim() : "";
  return s ? s : null;
}

export function pickRequestedBy(mintDTO: any, mintList: any): string | null {
  const v =
    mintDTO?.createdBy ??
    mintDTO?.requestedBy ??
    mintList?.createdBy ??
    mintList?.requestedBy ??
    null;

  const s = typeof v === "string" ? v.trim() : "";
  return s ? s : null;
}

export function pickScheduledBurnDate(mintDTO: any): string | null {
  const v = mintDTO?.scheduledBurnDate ?? mintDTO?.ScheduledBurnDate ?? null;
  if (!v) return null;
  const s = typeof v === "string" ? v.trim() : String(v);
  return s.trim() ? s.trim() : null;
}

export function pickMinted(mintDTO: any, mintList: any): boolean {
  if (typeof mintDTO?.minted === "boolean") return mintDTO.minted;
  if (mintDTO?.mintedAt) return true;

  if (typeof mintList?.minted === "boolean") return mintList.minted;
  if (mintList?.mintedAt) return true;

  return false;
}

export function pickProductBlueprintId(
  batch: any,
  productBlueprintIdById: Record<string, string | null>,
  pid: string,
): string | null {
  const v = batch?.productBlueprintId ?? batch?.productBlueprint?.id ?? null;

  const s1 = typeof v === "string" ? v.trim() : "";
  if (s1) return s1;

  const s2 = String(productBlueprintIdById?.[pid] ?? "").trim();
  return s2 ? s2 : null;
}

export function indexBatchesByProductionId(
  batches: InspectionBatchDTO[],
): Record<string, InspectionBatchDTO> {
  const out: Record<string, InspectionBatchDTO> = {};
  for (const b of batches ?? []) {
    const pid = normalizeProductionId(b);
    if (!pid) continue;
    out[pid] = b;
  }
  return out;
}

// ============================================================
// builder
// ============================================================

export function buildRowsJoined(
  productionIds: string[],
  productNameById: Record<string, string | null>,
  totalQuantityById: Record<string, number>,
  productBlueprintIdById: Record<string, string | null>,
  batchesById: Record<string, InspectionBatchDTO>,
  mintsDTOById: Record<string, MintDTO>,
  mintsListById: Record<string, MintListRowDTO>,
): ViewRow[] {
  const rows: ViewRow[] = [];

  for (const pid of productionIds ?? []) {
    const b = batchesById?.[pid] ?? null;

    const mintDTO: MintDTO | null = (mintsDTOById?.[pid] ?? null) as any;
    const mintList: MintListRowDTO | null = (mintsListById?.[pid] ?? null) as any;

    const inspStRaw = (b as any)?.status ?? null;
    const inspSt: InspectionStatus = (inspStRaw ?? "notYet") as any;

    const status = deriveMintStatusFromMintDTO(mintDTO);

    const tokenBlueprintId = pickTokenBlueprintId(mintDTO as any, mintList as any);
    const requestedBy = pickRequestedBy(mintDTO as any, mintList as any);

    const tokenName =
      asMaybeString((mintList as any)?.tokenName) ??
      asMaybeString((mintDTO as any)?.tokenName) ??
      null;

    const createdByName = asMaybeString((mintList as any)?.createdByName) ?? null;

    const mintedAt =
      (asMaybeString((mintDTO as any)?.mintedAt) ??
        asMaybeString((mintList as any)?.mintedAt) ??
        null) as string | null;

    const mintQuantity = Number((b as any)?.totalPassed ?? 0) || 0;

    const productionQuantity =
      Number(
        (b as any)?.quantity ??
          ((b as any)?.inspections?.length ?? 0) ??
          totalQuantityById?.[pid] ??
          0,
      ) || 0;

    const productName =
      (productNameById?.[pid] ?? null) ??
      (asMaybeString((b as any)?.productName) ?? null);

    const productBlueprintId = pickProductBlueprintId(
      b as any,
      productBlueprintIdById ?? {},
      pid,
    );

    const scheduledBurnDate = pickScheduledBurnDate(mintDTO as any);
    const minted = pickMinted(mintDTO as any, mintList as any);

    rows.push({
      id: pid,

      tokenName,
      productName,

      mintQuantity,
      productionQuantity,

      status,
      inspectionStatus: inspSt,

      createdByName,
      mintedAt,

      tokenBlueprintId,
      requestedBy,

      productBlueprintId,
      scheduledBurnDate,
      minted,

      statusLabel: inspectionStatusLabel(inspSt),
    });
  }

  return rows;
}
