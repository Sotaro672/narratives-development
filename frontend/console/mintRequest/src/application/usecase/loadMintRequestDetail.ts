// frontend/console/mintRequest/src/application/usecase/loadMintRequestDetail.ts

import type {
  InspectionBatchDTO,
  MintDTO,
} from "../../infrastructure/api/mintRequestApi";

import {
  fetchInspectionByProductionIdHTTP,
  fetchMintByInspectionIdHTTP,
  fetchProductBlueprintIdByProductionIdHTTP,
  fetchProductBlueprintPatchHTTP,
  fetchBrandsForMintHTTP,
  fetchTokenBlueprintsByBrandHTTP,
} from "../../infrastructure/repository";

// ✅ tokenBlueprint patch（Inventory 側の query endpoint を利用）
// NOTE: 3層厳密化する場合は infrastructure/adapter に隔離する想定だが、
// まずは互換優先で直接参照（段階移行用）
import {
  fetchTokenBlueprintPatchDTO,
  type TokenBlueprintPatchDTO,
} from "../../../../inventory/src/infrastructure/http/inventoryRepositoryHTTP";

// ============================================================
// Types (Detail model / DTO-ish)
// ============================================================

export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  brandName?: string | null;

  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: { type?: string | null } | null;

  assigneeId?: string | null;
};

export type BrandOption = {
  id: string;
  name: string;
};

export type TokenBlueprintOption = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

export type MintInfo = {
  id: string;

  brandId: string;
  tokenBlueprintId: string;

  createdBy: string;
  createdByName?: string | null;
  createdAt: string | null;

  minted: boolean;
  mintedAt?: string | null;
  onChainTxSignature?: string | null;
  scheduledBurnDate?: string | null;
};

// modelId 集計（検査結果の行）
export type ModelInspectionRow = {
  modelId: string;

  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;

  passedCount: number;
  totalCount: number;
};

export type MintRequestDetailModel = {
  requestId: string;

  batch: InspectionBatchDTO | null;
  mint: MintDTO | null;

  mintInfo: MintInfo | null;

  productBlueprintId: string | null;
  productBlueprintPatch: ProductBlueprintPatchDTO | null;

  brandOptions: BrandOption[];
  tokenBlueprintOptions: TokenBlueprintOption[];

  tokenBlueprintPatch: TokenBlueprintPatchDTO | null;

  modelRows: ModelInspectionRow[];
};

// ============================================================
// Small helpers
// ============================================================

export function asNonEmptyString(v: any): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

export function asMaybeISO(v: any): string {
  if (!v) return "";
  if (typeof v === "string") return v;
  if (v instanceof Date) return v.toISOString();
  return String(v);
}

function asNumberOrNull(v: any): number | null {
  if (typeof v === "number") return Number.isFinite(v) ? v : null;
  const n = Number(v);
  return Number.isFinite(n) ? n : null;
}

function isPassedResult(v: any): boolean {
  const s = asNonEmptyString(v).toLowerCase();
  return s === "passed";
}

// -------------------------------
// productBlueprintId 抽出/解決
// -------------------------------

export function extractProductBlueprintIdFromBatch(batch: any): string {
  if (!batch) return "";
  const v = batch.productBlueprintId ?? batch.productBlueprint?.id ?? "";
  return asNonEmptyString(v);
}

export async function resolveProductBlueprintIdByRequestId(
  requestId: string,
  batch: InspectionBatchDTO | null,
): Promise<string> {
  const rid = String(requestId ?? "").trim();
  if (!rid) return "";

  const pbFromBatch = extractProductBlueprintIdFromBatch(batch as any);
  if (pbFromBatch) return pbFromBatch;

  const pbFromProduction = await fetchProductBlueprintIdByProductionIdHTTP(rid).catch(
    () => null,
  );
  return asNonEmptyString(pbFromProduction);
}

// -------------------------------
// model rows（modelId 集計のみ）
// -------------------------------

export function buildModelRowsFromBatch(
  batch: InspectionBatchDTO | null,
): ModelInspectionRow[] {
  const inspections: any[] = Array.isArray((batch as any)?.inspections)
    ? ((batch as any).inspections as any[])
    : [];

  const agg = new Map<string, { modelId: string; passed: number; total: number }>();

  for (const it of inspections) {
    const modelId = asNonEmptyString(it?.modelId);
    if (!modelId) continue;

    const prev = agg.get(modelId) ?? { modelId, passed: 0, total: 0 };
    prev.total += 1;

    const result = it?.inspectionResult ?? null;
    if (isPassedResult(result)) prev.passed += 1;

    agg.set(modelId, prev);
  }

  const rows: ModelInspectionRow[] = Array.from(agg.values()).map((g) => ({
    modelId: g.modelId,
    modelNumber: null,
    size: null,
    colorName: null,
    rgb: null,
    passedCount: g.passed,
    totalCount: g.total,
  }));

  rows.sort((a, b) => a.modelId.localeCompare(b.modelId));
  return rows;
}

// -------------------------------
// MintInfo 解決（mintDTO 優先）
// -------------------------------

export function extractMintInfoFromMintDTO(m: any): MintInfo | null {
  if (!m) return null;

  const id = asNonEmptyString(m.id ?? m.mintId);
  if (!id) return null;

  const tokenBlueprintId = asNonEmptyString(m.tokenBlueprintId);
  const brandId = asNonEmptyString(m.brandId);

  const createdBy = asNonEmptyString(m.createdBy);
  const createdByName = asNonEmptyString(m.createdByName);

  const createdAtStr = asNonEmptyString(asMaybeISO(m.createdAt));
  const createdAt = createdAtStr ? createdAtStr : null;

  const mintedAtStr = asNonEmptyString(asMaybeISO(m.mintedAt));
  const minted = typeof m.minted === "boolean" ? m.minted : Boolean(mintedAtStr);

  const onChainTxSignature = asNonEmptyString(m.onChainTxSignature);
  const scheduledBurnDate = asNonEmptyString(asMaybeISO(m.scheduledBurnDate));

  return {
    id,
    brandId,
    tokenBlueprintId,
    createdBy,
    createdByName: createdByName ? createdByName : null,
    createdAt,
    minted,
    mintedAt: mintedAtStr ? mintedAtStr : null,
    onChainTxSignature: onChainTxSignature ? onChainTxSignature : null,
    scheduledBurnDate: scheduledBurnDate ? scheduledBurnDate : null,
  };
}

export function extractMintInfoFromBatch(batch: any): MintInfo | null {
  if (!batch) return null;

  const mintObj = batch.mint ?? batch.mintRequest ?? null;
  if (!mintObj) return null;

  return extractMintInfoFromMintDTO(mintObj);
}

// ============================================================
// Usecase: Load MintRequest Detail
// ============================================================

export async function loadMintRequestDetail(
  requestId: string,
): Promise<MintRequestDetailModel> {
  const rid = String(requestId ?? "").trim();

  if (!rid) {
    return {
      requestId: "",
      batch: null,
      mint: null,
      mintInfo: null,
      productBlueprintId: null,
      productBlueprintPatch: null,
      brandOptions: [],
      tokenBlueprintOptions: [],
      tokenBlueprintPatch: null,
      modelRows: [],
    };
  }

  // 1) inspection(batch) + mint
  const [batch, mint] = await Promise.all([
    fetchInspectionByProductionIdHTTP(rid).catch(() => null),
    fetchMintByInspectionIdHTTP(rid).catch(() => null),
  ]);

  // 2) productBlueprintId
  const productBlueprintIdStr = await resolveProductBlueprintIdByRequestId(
    rid,
    batch as any,
  ).catch(() => "");
  const productBlueprintId = productBlueprintIdStr ? productBlueprintIdStr : null;

  // 3) productBlueprintPatch
  const productBlueprintPatch: ProductBlueprintPatchDTO | null = productBlueprintId
    ? await fetchProductBlueprintPatchHTTP(productBlueprintId).catch(() => null)
    : null;

  // 4) options: brands / tokenBlueprints
  const brandsRaw = await fetchBrandsForMintHTTP().catch(() => []);
  const brandOptions: BrandOption[] = (brandsRaw ?? [])
    .map((b: any) => ({
      id: asNonEmptyString(b?.id ?? b?.ID),
      name: asNonEmptyString(b?.name ?? b?.Name),
    }))
    .filter((b) => b.id && b.name);

  const selectedBrandId =
    asNonEmptyString((mint as any)?.brandId) ||
    asNonEmptyString((productBlueprintPatch as any)?.brandId) ||
    "";

  const tokenBlueprintsRaw = selectedBrandId
    ? await fetchTokenBlueprintsByBrandHTTP(selectedBrandId).catch(() => [])
    : [];

  const tokenBlueprintOptions: TokenBlueprintOption[] = (tokenBlueprintsRaw ?? [])
    .map((tb: any) => ({
      id: asNonEmptyString(tb?.id ?? tb?.ID),
      name: asNonEmptyString(tb?.name ?? tb?.Name),
      symbol: asNonEmptyString(tb?.symbol ?? tb?.Symbol),
      iconUrl: asNonEmptyString(tb?.iconUrl ?? tb?.IconUrl) || undefined,
    }))
    .filter((tb) => tb.id && tb.name && tb.symbol);

  // 5) inventory tokenBlueprintPatch（あれば）
  const tokenBlueprintId =
    asNonEmptyString((mint as any)?.tokenBlueprintId) ||
    asNonEmptyString((batch as any)?.tokenBlueprintId) ||
    "";

  const tokenBlueprintPatch: TokenBlueprintPatchDTO | null = tokenBlueprintId
    ? await fetchTokenBlueprintPatchDTO(tokenBlueprintId).catch(() => null)
    : null;

  // 6) model rows
  const modelRows = buildModelRowsFromBatch(batch);

  // 7) mintInfo（mintDTO 優先、なければ batch 内の埋め込みを拾う）
  const mintInfo =
    extractMintInfoFromMintDTO(mint as any) ?? extractMintInfoFromBatch(batch as any);

  return {
    requestId: rid,
    batch: (batch ?? null) as any,
    mint: (mint ?? null) as any,
    mintInfo: mintInfo ?? null,
    productBlueprintId,
    productBlueprintPatch: (productBlueprintPatch ?? null) as any,
    brandOptions,
    tokenBlueprintOptions,
    tokenBlueprintPatch: (tokenBlueprintPatch ?? null) as any,
    modelRows,
  };
}
