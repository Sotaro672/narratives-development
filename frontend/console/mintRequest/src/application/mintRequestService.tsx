// frontend/console/mintRequest/src/application/mintRequestService.tsx
import type { InspectionStatus } from "../domain/entity/inspections";
import {
  fetchInspectionBatches, // fallback (legacy)
  type InspectionBatchDTO,
  type MintListRowDTO,
  type MintDTO,
} from "../infrastructure/api/mintRequestApi";

// ✅ repository から取得する（API_BASE / token / normalize を集約）
import {
  listMintsByInspectionIDsHTTP,
  fetchMintsByInspectionIdsHTTP, // view=dto 想定
  fetchMintByInspectionIdHTTP,
  fetchInspectionBatchesHTTP,
  fetchInspectionBatchesByProductionIdsHTTP,

  // ✅ detail 用（hook 直呼びをやめて、service に集約）
  fetchInspectionByProductionIdHTTP,
  fetchProductBlueprintPatchHTTP,
  fetchBrandsForMintHTTP,
  fetchTokenBlueprintsByBrandHTTP,
  postMintRequestHTTP,
  fetchProductBlueprintIdByProductionIdHTTP,
} from "../infrastructure/repository";

// ✅ tokenBlueprint patch（Inventory 側の query endpoint を利用）
import {
  fetchTokenBlueprintPatchDTO,
  type TokenBlueprintPatchDTO,
} from "../../../inventory/src/infrastructure/http/inventoryRepositoryHTTP";

import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// ============================================================
// Types (ViewModel for MintRequestManagement)
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

  // ✅ detail で必要になるキーを保持
  tokenBlueprintId: string | null;
  requestedBy: string | null;

  // ✅ 追加: detail で必要
  productBlueprintId: string | null;
  scheduledBurnDate: string | null;
  minted: boolean;

  statusLabel: string; // 画面表示用（ここでは検査ステータス）
};

// ============================================================
// API base / fetch helpers (productions はここだけで取得)
// ※ mintRequestRepositoryHTTP.ts に /productions 系があるなら、そちらへ寄せてもOK
// ============================================================

const API_BASE = String((import.meta as any)?.env?.VITE_BACKEND_BASE_URL ?? "")
  .trim()
  .replace(/\/$/, "");

async function fetchJsonWithAuth<T>(path: string): Promise<T> {
  if (!API_BASE) {
    throw new Error("VITE_BACKEND_BASE_URL is empty");
  }

  const user = auth.currentUser;
  const token = user ? await user.getIdToken() : "";

  const url = `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  });

  const text = await res.text();

  if (!res.ok) {
    throw new Error(`HTTP ${res.status} ${url}: ${text?.slice?.(0, 200) ?? ""}`);
  }

  try {
    return JSON.parse(text) as T;
  } catch (e: any) {
    throw new Error(`Invalid JSON from ${url}: ${e?.message ?? e}`);
  }
}

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

/**
 * ✅ 方針
 * - 新コードでは lowerCamel を正とする
 * - ただし「id vs productionId」「TokenBlueprintID vs tokenBlueprintId」など
 *   実運用で残りがちな差分だけ最小限残す
 */

function normalizeProductionId(v: any): string {
  // inspectionBatch は productionId が正。念のため inspectionId/id も許容（最小限）
  return String(v?.productionId ?? v?.inspectionId ?? v?.id ?? "").trim();
}

function normalizeProductionIdFromProductionListItem(v: any): string {
  // /productions は id or productionId を想定（ネスト構造は切り捨て）
  return String(v?.productionId ?? v?.id ?? "").trim();
}

function normalizeProductNameFromProductionListItem(v: any): string | null {
  const s = String(v?.productName ?? v?.name ?? "").trim();
  return s ? s : null;
}

function normalizeTotalQuantityFromProductionListItem(v: any): number {
  const n = Number(v?.totalQuantity ?? v?.quantity ?? 0) || 0;
  return n > 0 ? n : 0;
}

function normalizeProductBlueprintIdFromProductionListItem(
  v: any,
): string | null {
  // /productions は productBlueprintId を正。念のため productBlueprint?.id だけ残す
  const s = String(v?.productBlueprintId ?? v?.productBlueprint?.id ?? "").trim();
  return s ? s : null;
}

function deriveMintStatusFromMintDTO(
  mint: MintDTO | null,
): MintRequestRowStatus {
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

// ============================================================
// New flow: inspectionsDTO（芯） + mintsDTO（肉付け）を productionId で join
// ============================================================

type ProductionIndex = {
  productionIds: string[];
  productNameById: Record<string, string | null>;
  totalQuantityById: Record<string, number>;
  productBlueprintIdById: Record<string, string | null>;
};

async function fetchProductionIndex(): Promise<ProductionIndex> {
  const items = await fetchJsonWithAuth<any[]>("/productions");

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
    productBlueprintIdById[pid] =
      normalizeProductBlueprintIdFromProductionListItem(it);
  }

  return {
    productionIds,
    productNameById,
    totalQuantityById,
    productBlueprintIdById,
  };
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

    const inspSt = ((b as any)?.status ?? null) as any;
    const status = deriveMintStatusFromMintDTO(mintDTO);

    const tokenBlueprintId = pickTokenBlueprintId(mintDTO as any, mintList as any);
    const requestedBy = pickRequestedBy(mintDTO as any, mintList as any);

    // ✅ tokenName は “名前解決済み” を優先（list row）
    const tokenName =
      (mintList as any)?.tokenName ?? (mintDTO as any)?.tokenName ?? null;

    const createdByName = (mintList as any)?.createdByName ?? null;

    const mintedAt =
      ((mintDTO as any)?.mintedAt ?? (mintList as any)?.mintedAt ?? null) as
        | string
        | null;

    const mintQuantity = Number((b as any)?.totalPassed ?? 0) || 0;

    const productionQuantity =
      Number(
        (b as any)?.quantity ??
          ((b as any)?.inspections?.length ?? 0) ??
          totalQuantityById?.[pid] ??
          0,
      ) || 0;

    const productName = (productNameById?.[pid] ?? null) as string | null;

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
      inspectionStatus: (inspSt ?? "inspecting") as InspectionStatus,

      createdByName,
      mintedAt,

      tokenBlueprintId,
      requestedBy,

      productBlueprintId,
      scheduledBurnDate,
      minted,

      statusLabel: inspectionStatusLabel(inspSt as any),
    });
  }

  return rows;
}

// ============================================================
// Service: MintRequestManagement (list screen)
// ============================================================

export async function loadMintRequestManagementRows(): Promise<ViewRow[]> {
  // 0) productionIds（芯）
  const {
    productionIds,
    productNameById,
    totalQuantityById,
    productBlueprintIdById,
  } = await fetchProductionIndex();

  // 互換 fallback
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

  if (effectiveProductionIds.length === 0) {
    return [];
  }

  // 1) inspectionsDTO（芯）
  let batches: InspectionBatchDTO[] = [];
  try {
    if (productionIds.length > 0) {
      batches = await fetchInspectionBatchesByProductionIdsHTTP(
        effectiveProductionIds,
      );
    } else {
      batches = await fetchInspectionBatchesHTTP();
    }
  } catch {
    batches = [];
  }

  const batchesById = indexBatchesByProductionId(batches);

  // 2) mintsDTO（肉付け）
  let mintsDTOById: Record<string, MintDTO> = {};
  try {
    mintsDTOById = (await fetchMintsByInspectionIdsHTTP(
      effectiveProductionIds,
    )) as any;
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
  const rows = buildRowsJoined(
    effectiveProductionIds,
    productNameById ?? {},
    totalQuantityById ?? {},
    productBlueprintIdById ?? {},
    batchesById ?? {},
    mintsDTOById ?? {},
    mintsListById ?? {},
  );

  return rows;
}

// ============================================================
// Service: MintDTO fetch (full DTO)
// ============================================================

export async function loadMintsDTOMapByInspectionIds(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const m = await fetchMintsByInspectionIdsHTTP(ids);
  return (m ?? {}) as Record<string, MintDTO>;
}

export async function loadMintDTOByInspectionId(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) return null;

  const m = await fetchMintByInspectionIdHTTP(iid);
  return (m ?? null) as MintDTO | null;
}

// ============================================================
// ====================== Detail (useMintRequestDetail) ======================
// hook から「純粋関数 / 型 / データ取得」を移譲して集約する
// ============================================================

// -------------------------------
// Local DTOs (detail)
// -------------------------------

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

export type BrandForMintDTO = {
  id: string;
  name: string;
};

export type TokenBlueprintForMintDTO = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

export type ProductBlueprintCardViewModel = {
  productName?: string;
  brand?: string; // ✅ 表示用（brandNameのみ）
  itemType?: string;
  fit?: string;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;
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

export type TokenBlueprintCardViewModel = {
  id: string;
  name: string;
  symbol: string;

  // ⚠️ brandId は UI 表示に使わせない（揺れ防止のため空文字を渡す）
  brandId: string;

  // ✅ UI 表示は brandName のみに統一
  brandName: string;

  description: string;
  iconUrl?: string;
  isEditMode: boolean;
  brandOptions: { id: string; name: string }[];
};

export type TokenBlueprintCardHandlers = {
  onPreview: () => void;
};

export type MintInfo = {
  id: string;

  brandId: string;
  tokenBlueprintId: string;

  createdBy: string;
  createdByName?: string | null; // ★表示はこれを優先
  createdAt: string | null;

  minted: boolean;
  mintedAt?: string | null;
  onChainTxSignature?: string | null;
  scheduledBurnDate?: string | null;
};

// ============================================================
// ✅ model rows（まずは modelId 集計のみ）
// ============================================================

export type MintModelMetaEntry = {
  modelNumber?: string | null;
  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;
};

export type ModelInspectionRow = {
  modelId: string;

  // 現状は未解決（後で /models 側から解決予定）
  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;

  passedCount: number; // 合格数
  totalCount: number; // 生産数（このモデルの対象件数）
};

export function asNonEmptyString(v: any): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

export function asMaybeISO(v: any): string {
  if (!v) return "";
  if (typeof v === "string") return v;
  if (v instanceof Date) return v.toISOString();
  return String(v);
}

export function safeDateTimeLabelJa(
  v: string | null | undefined,
  fallback: string,
) {
  const s = asNonEmptyString(v);
  if (!s) return fallback;
  const t = Date.parse(s);
  if (Number.isNaN(t)) return s; // 解析不可なら生文字
  return new Date(t).toLocaleString("ja-JP");
}

export function safeDateLabelJa(v: string | null | undefined, fallback: string) {
  const s = asNonEmptyString(v);
  if (!s) return fallback;
  const t = Date.parse(s);
  if (Number.isNaN(t)) {
    // "YYYY-MM-DD" などはそのまま出したいケースがあるので生文字返す
    return s;
  }
  return new Date(t).toLocaleDateString("ja-JP");
}

// -------------------------------
// ★ productBlueprintId 抽出/解決
// -------------------------------

export function extractProductBlueprintIdFromBatch(batch: any): string {
  if (!batch) return "";
  const v = batch.productBlueprintId ?? batch.productBlueprint?.id ?? "";
  return asNonEmptyString(v);
}

function isPassedResult(v: any): boolean {
  const s = asNonEmptyString(v).toLowerCase();
  return s === "passed";
}

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
// data loaders (detail)
// -------------------------------

export async function fetchInspectionAndMintByRequestId(requestId: string): Promise<{
  batch: InspectionBatchDTO | null;
  mint: MintDTO | null;
}> {
  const rid = String(requestId ?? "").trim();
  if (!rid) return { batch: null, mint: null };

  const [batch, mint] = await Promise.all([
    fetchInspectionByProductionIdHTTP(rid),
    fetchMintByInspectionIdHTTP(rid).catch(() => null),
  ]);

  return { batch: (batch ?? null) as any, mint: (mint ?? null) as any };
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

export async function fetchProductBlueprintPatchById(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const id = String(productBlueprintId ?? "").trim();
  if (!id) return null;

  const patch = await fetchProductBlueprintPatchHTTP(id);
  return (patch ?? null) as any;
}

export async function fetchBrandOptionsForMint(): Promise<BrandOption[]> {
  const brands = await fetchBrandsForMintHTTP();
  return (brands ?? []).map((b: any) => ({
    id: String(b?.id ?? "").trim(),
    name: String(b?.name ?? "").trim(),
  }));
}

export async function fetchTokenBlueprintOptionsByBrand(
  brandId: string,
): Promise<TokenBlueprintOption[]> {
  const id = String(brandId ?? "").trim();
  if (!id) return [];

  const list = await fetchTokenBlueprintsByBrandHTTP(id);

  return (list ?? []).map((tb: any) => ({
    id: String(tb?.id ?? "").trim(),
    name: String(tb?.name ?? "").trim(),
    symbol: String(tb?.symbol ?? "").trim(),
    iconUrl: asNonEmptyString(tb?.iconUrl) || undefined,
  }));
}

/**
 * 互換: 現状は個別 TokenBlueprint 詳細 API がないため undefined を返す。
 */
export function resolveBlueprintForMintRequest(_requestId?: string) {
  return undefined;
}

// -------------------------------
// ★ MintInfo 解決（mintDTO 優先）
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

export async function fetchTokenBlueprintPatchById(
  tokenBlueprintId: string,
): Promise<TokenBlueprintPatchDTO | null> {
  const tbId = String(tokenBlueprintId ?? "").trim();
  if (!tbId) return null;

  const p = await fetchTokenBlueprintPatchDTO(tbId);
  return (p ?? null) as any;
}

export async function submitMintRequestAndRefresh(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
): Promise<{
  updatedBatch: InspectionBatchDTO | null;
  refreshedMint: MintDTO | null;
}> {
  const pid = String(productionId ?? "").trim();
  const tbId = String(tokenBlueprintId ?? "").trim();
  if (!pid || !tbId) return { updatedBatch: null, refreshedMint: null };

  const updated = await postMintRequestHTTP(pid, tbId, scheduledBurnDate);

  let refreshed: MintDTO | null = null;
  try {
    refreshed = await fetchMintByInspectionIdHTTP(pid).catch(() => null);
  } catch {
    refreshed = null;
  }

  return {
    updatedBatch: (updated ?? null) as any,
    refreshedMint: (refreshed ?? null) as any,
  };
}
