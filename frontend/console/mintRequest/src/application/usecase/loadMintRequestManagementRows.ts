// frontend/console/mintRequest/src/application/usecase/loadMintRequestManagementRows.ts

import type { InspectionStatus } from "../../domain/entity/inspections";
import {
  fetchInspectionBatches, // fallback (legacy)
  type InspectionBatchDTO,
  type MintListRowDTO,
  type MintDTO,
} from "../../infrastructure/api/mintRequestApi";

import {
  listMintsByInspectionIDsHTTP,
  fetchMintsByInspectionIdsHTTP, // view=dto 想定
  fetchInspectionBatchesHTTP,
  fetchInspectionBatchesByProductionIdsHTTP,
} from "../../infrastructure/repository";

import * as repoNS from "../../infrastructure/repository";

// ============================================================
// Types (ViewModel-like row for Management list)
// ※ 現状の画面が期待している形を維持（段階移行用）
// ============================================================

export type MintRequestRowStatus = "planning" | "requested" | "minted";

export type ViewRow = {
  id: string; // = productionId (= inspectionId 扱い)

  tokenName: string | null;
  productName: string | null;

  mintQuantity: number; // = inspection.totalPassed
  productionQuantity: number; // = inspection.quantity (fallback: production.totalQuantity)

  status: MintRequestRowStatus; // = mint の有無・mintedAt で判定
  inspectionStatus: InspectionStatus; // = inspection.status

  createdByName: string | null;
  mintedAt: string | null;

  // detail で必要になるキー
  tokenBlueprintId: string | null;
  requestedBy: string | null;

  // detail で必要
  productBlueprintId: string | null;
  scheduledBurnDate: string | null;
  minted: boolean;

  statusLabel: string; // 画面表示用（ここでは検査ステータス）
};

// ============================================================
// Pure helpers (list)
// ============================================================

const inspectionStatusLabel = (
  s: InspectionStatus | null | undefined,
): string => {
  switch (s) {
    case "inspecting":
      return "検査中";
    case "completed":
      return "検査完了";
    default:
      return "未検査";
  }
};

function asTrimmedString(v: any): string {
  return typeof v === "string" ? v.trim() : String(v ?? "").trim();
}

function asMaybeString(v: any): string | null {
  const s = asTrimmedString(v);
  return s ? s : null;
}

function normalizeProductionId(v: any): string {
  // inspectionBatch は productionId が正。念のため inspectionId/id も許容（最小限）
  return String(v?.productionId ?? v?.inspectionId ?? v?.id ?? "").trim();
}

function normalizeProductionIdFromProductionListItem(v: any): string {
  // /productions は id or productionId を想定（ネスト構造は切り捨て）
  return String(v?.productionId ?? v?.ProductionId ?? v?.id ?? v?.ID ?? "").trim();
}

function normalizeProductNameFromProductionListItem(v: any): string | null {
  const s = String(v?.productName ?? v?.ProductName ?? v?.name ?? v?.Name ?? "").trim();
  return s ? s : null;
}

function normalizeTotalQuantityFromProductionListItem(v: any): number {
  const n = Number(v?.totalQuantity ?? v?.TotalQuantity ?? v?.quantity ?? v?.Quantity ?? 0) || 0;
  return n > 0 ? n : 0;
}

function normalizeProductBlueprintIdFromProductionListItem(v: any): string | null {
  // /productions は productBlueprintId を正。念のため productBlueprint?.id だけ残す
  const s = String(
    v?.productBlueprintId ??
      v?.ProductBlueprintId ??
      v?.productBlueprintID ??
      v?.ProductBlueprintID ??
      v?.productBlueprint?.id ??
      v?.productBlueprint?.ID ??
      "",
  ).trim();
  return s ? s : null;
}

function deriveMintStatusFromMintDTO(mint: MintDTO | null): MintRequestRowStatus {
  if (!mint) return "planning";
  if ((mint as any)?.mintedAt) return "minted";
  if ((mint as any)?.minted === true) return "minted";
  return "requested";
}

function pickTokenBlueprintId(mintDTO: any, mintList: any): string | null {
  const v =
    mintDTO?.tokenBlueprintId ??
    mintDTO?.TokenBlueprintID ??
    mintList?.tokenBlueprintId ??
    mintList?.TokenBlueprintID ??
    null;

  const s = typeof v === "string" ? v.trim() : "";
  return s ? s : null;
}

function pickRequestedBy(mintDTO: any, mintList: any): string | null {
  // createdBy が正。requestedBy は旧名の可能性があるので最小限残す
  const v =
    mintDTO?.createdBy ??
    mintDTO?.requestedBy ??
    mintList?.createdBy ??
    mintList?.requestedBy ??
    null;

  const s = typeof v === "string" ? v.trim() : "";
  return s ? s : null;
}

function pickScheduledBurnDate(mintDTO: any): string | null {
  const v = mintDTO?.scheduledBurnDate ?? mintDTO?.ScheduledBurnDate ?? null;
  if (!v) return null;
  const s = typeof v === "string" ? v.trim() : String(v);
  return s.trim() ? s.trim() : null;
}

function pickMinted(mintDTO: any, mintList: any): boolean {
  if (typeof mintDTO?.minted === "boolean") return mintDTO.minted;
  if (mintDTO?.mintedAt) return true;

  if (typeof mintList?.minted === "boolean") return mintList.minted;
  if (mintList?.mintedAt) return true;

  return false;
}

function pickProductBlueprintId(
  batch: any,
  productBlueprintIdById: Record<string, string | null>,
  pid: string,
): string | null {
  // inspectionBatch は productBlueprintId が正。念のため productBlueprint?.id を許容
  const v = batch?.productBlueprintId ?? batch?.productBlueprint?.id ?? null;

  const s1 = typeof v === "string" ? v.trim() : "";
  if (s1) return s1;

  const s2 = String(productBlueprintIdById?.[pid] ?? "").trim();
  return s2 ? s2 : null;
}

function indexBatchesByProductionId(
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
// productions index (optional: repository 側に実装がある場合だけ使う)
// ============================================================

type ProductionIndex = {
  productionIds: string[];
  productNameById: Record<string, string | null>;
  totalQuantityById: Record<string, number>;
  productBlueprintIdById: Record<string, string | null>;
};

function normalizeProductionsPayload(json: any): any[] {
  if (Array.isArray(json)) return json;
  const items =
    json?.items ??
    json?.Items ??
    json?.productions ??
    json?.Productions ??
    null;
  return Array.isArray(items) ? items : [];
}

async function fetchProductionsAny(): Promise<any[]> {
  const anyRepo = repoNS as any;

  const candidates = [
    "fetchProductionsHTTP",
    "fetchProductionsForCurrentCompanyHTTP",
    "listProductionsHTTP",
    "fetchProductions",
  ];

  for (const name of candidates) {
    const fn = anyRepo?.[name];
    if (typeof fn === "function") {
      const res = await fn();
      return normalizeProductionsPayload(res);
    }
  }

  // 次善：ID だけ取得できる実装があればそれを使う
  const idsFn =
    anyRepo?.fetchProductionIdsForCurrentCompanyHTTP ??
    anyRepo?.listProductionIdsHTTP ??
    null;

  if (typeof idsFn === "function") {
    const ids: string[] = (await idsFn()) ?? [];
    return (ids ?? []).map((id) => ({ productionId: id }));
  }

  return [];
}

async function fetchProductionIndex(): Promise<ProductionIndex> {
  const items = await fetchProductionsAny();

  const productionIds: string[] = [];
  const seen = new Set<string>();

  const productNameById: Record<string, string | null> = {};
  const totalQuantityById: Record<string, number> = {};
  const productBlueprintIdById: Record<string, string | null> = {};

  for (const it of items ?? []) {
    const pid = normalizeProductionIdFromProductionListItem(it);
    if (!pid || seen.has(pid)) continue;
    seen.add(pid);
    productionIds.push(pid);

    productNameById[pid] = normalizeProductNameFromProductionListItem(it);
    totalQuantityById[pid] = normalizeTotalQuantityFromProductionListItem(it);
    productBlueprintIdById[pid] = normalizeProductBlueprintIdFromProductionListItem(it);
  }

  return { productionIds, productNameById, totalQuantityById, productBlueprintIdById };
}

// ============================================================
// join builder
// ============================================================

function buildRowsJoined(
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

    // tokenName は “名前解決済み” を優先（list row）
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

// ============================================================
// Usecase: MintRequestManagement (list screen)
// ============================================================

export async function loadMintRequestManagementRows(): Promise<ViewRow[]> {
  // 0) productions（あれば全 production を出すための “芯”）
  const {
    productionIds,
    productNameById,
    totalQuantityById,
    productBlueprintIdById,
  } = await fetchProductionIndex();

  // fallback: productions が取れない場合は inspections から id を作る（旧互換）
  let effectiveProductionIds = productionIds;
  if (effectiveProductionIds.length === 0) {
    try {
      const legacyBatches = await fetchInspectionBatches();
      const legacyIds = (legacyBatches ?? [])
        .map((b) => normalizeProductionId(b))
        .filter((s) => !!s);

      effectiveProductionIds = legacyIds;
    } catch {
      effectiveProductionIds = [];
    }
  }

  if (effectiveProductionIds.length === 0) return [];

  // 1) inspections（芯）
  let batches: InspectionBatchDTO[] = [];
  try {
    // productionIds が取れていれば ids 指定で取得を優先（無駄な取得を抑える）
    if (productionIds.length > 0) {
      batches = await fetchInspectionBatchesByProductionIdsHTTP(effectiveProductionIds);
    } else {
      // 取得手段が ids しかない場合もあるので、まず ids 指定を試す
      try {
        batches = await fetchInspectionBatchesByProductionIdsHTTP(effectiveProductionIds);
      } catch {
        batches = await fetchInspectionBatchesHTTP();
      }
    }
  } catch {
    batches = [];
  }

  const batchesById = indexBatchesByProductionId(batches);

  // 2) mints DTO（肉付け）
  let mintsDTOById: Record<string, MintDTO> = {};
  try {
    mintsDTOById = (await fetchMintsByInspectionIdsHTTP(effectiveProductionIds)) as any;
  } catch {
    mintsDTOById = {};
  }

  // 3) 表示用 list row（tokenName / createdByName）
  let mintsListById: Record<string, MintListRowDTO> = {};
  try {
    mintsListById = await listMintsByInspectionIDsHTTP(effectiveProductionIds);
  } catch {
    mintsListById = {};
  }

  // 4) join
  return buildRowsJoined(
    effectiveProductionIds,
    productNameById ?? {},
    totalQuantityById ?? {},
    productBlueprintIdById ?? {},
    batchesById ?? {},
    mintsDTOById ?? {},
    mintsListById ?? {},
  );
}
