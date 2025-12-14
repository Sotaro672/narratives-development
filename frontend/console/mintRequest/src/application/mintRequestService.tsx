// frontend\console\mintRequest\src\application\mintRequestService.tsx
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
} from "../infrastructure/repository/mintRequestRepositoryHTTP";

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
// Debug logger
// ============================================================

const log = (...args: any[]) => {
  // eslint-disable-next-line no-console
  console.log("[mintRequest/mintRequestService]", ...args);
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
  log("fetchJsonWithAuth", { url, hasToken: !!token });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  });

  const text = await res.text();

  log("fetchJsonWithAuth response", {
    url,
    status: res.status,
    ok: res.ok,
    bodySample: text?.slice?.(0, 300),
  });

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
// Pure helpers
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

function normalizeProductionId(v: any): string {
  return String(v?.productionId ?? v?.inspectionId ?? v?.id ?? "").trim();
}

function normalizeProductionIdFromProductionListItem(v: any): string {
  return String(
    v?.productionId ??
      v?.id ??
      v?.production?.id ??
      v?.production?.productionId ??
      "",
  ).trim();
}

function normalizeProductNameFromProductionListItem(v: any): string | null {
  const s = String(
    v?.productName ??
      v?.production?.productName ??
      v?.productionName ??
      v?.production?.name ??
      "",
  ).trim();
  return s ? s : null;
}

function normalizeTotalQuantityFromProductionListItem(v: any): number {
  const n =
    Number(
      v?.totalQuantity ??
        v?.production?.totalQuantity ??
        v?.quantity ??
        0,
    ) || 0;
  return n > 0 ? n : 0;
}

function normalizeProductBlueprintIdFromProductionListItem(v: any): string | null {
  const s = String(
    v?.productBlueprintId ??
      v?.productBlueprintID ??
      v?.ProductBlueprintId ??
      v?.ProductBlueprintID ??
      v?.production?.productBlueprintId ??
      v?.production?.productBlueprintID ??
      v?.production?.ProductBlueprintId ??
      v?.production?.ProductBlueprintID ??
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
    mintDTO?.tokenBlueprintID ??
    mintDTO?.token_blueprint_id ??
    mintList?.tokenBlueprintId ??
    mintList?.tokenBlueprintID ??
    mintList?.tokenBlueprint ??
    null;

  const s = typeof v === "string" ? v.trim() : "";
  return s ? s : null;
}

function pickRequestedBy(mintDTO: any, mintList: any): string | null {
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
  const v =
    batch?.productBlueprintId ??
    batch?.productBlueprintID ??
    batch?.ProductBlueprintID ??
    batch?.ProductBlueprintId ??
    batch?.productBlueprint?.id ??
    batch?.productBlueprint?.ID ??
    null;

  const s1 = typeof v === "string" ? v.trim() : "";
  if (s1) return s1;

  const s2 = String(productBlueprintIdById?.[pid] ?? "").trim();
  return s2 ? s2 : null;
}

// ============================================================
// New flow: inspectionsDTO（芯） + mintsDTO（肉付け）を productionId で join
// ※ inspections は repository 経由で取得する（ここが今回の変更点）
// ============================================================

type ProductionIndex = {
  productionIds: string[];
  productNameById: Record<string, string | null>;
  totalQuantityById: Record<string, number>;
  productBlueprintIdById: Record<string, string | null>;
};

async function fetchProductionIndex(): Promise<ProductionIndex> {
  const items = await fetchJsonWithAuth<any[]>("/productions");

  log(
    "/productions fetched",
    "length=",
    (items ?? []).length,
    "sample[0]=",
    (items ?? [])[0],
  );

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

  log(
    "productionIds",
    "len=",
    productionIds.length,
    "sample[0..4]=",
    productionIds.slice(0, 5),
  );

  log(
    "productBlueprintIdById(sample)",
    productionIds.slice(0, 3).map((id) => ({ id, pbId: productBlueprintIdById[id] })),
  );

  return { productionIds, productNameById, totalQuantityById, productBlueprintIdById };
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

    // ✅ tokenName は “名前解決済み” を優先
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
  log("load start", { API_BASE });

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
      log("fallback productionIds from legacy inspections", {
        len: legacyIds.length,
        sample: legacyIds.slice(0, 5),
      });
    } catch (e: any) {
      log("fallback fetchInspectionBatches failed", e?.message ?? e);
      effectiveProductionIds = [];
    }
  }

  if (effectiveProductionIds.length === 0) {
    log("no productionIds -> return []");
    return [];
  }

  // 1) inspectionsDTO（芯）
  // ✅ ここを repository 経由に変更（API_BASE/トークン/normalize を repository に集約）
  // - productionIds が取れているなら「byProductionIds」を優先
  // - 取れていない fallback 時は fetchInspectionBatchesHTTP()（内部で /productions -> /mint/inspections をやる）
  let batches: InspectionBatchDTO[] = [];
  try {
    if (productionIds.length > 0) {
      batches = await fetchInspectionBatchesByProductionIdsHTTP(
        effectiveProductionIds,
      );
    } else {
      batches = await fetchInspectionBatchesHTTP();
    }
    log(
      "inspections fetched via repository",
      "len=",
      (batches ?? []).length,
      "sample[0]=",
      (batches ?? [])[0],
    );
  } catch (e: any) {
    log("fetchInspectionBatches via repository failed", e?.message ?? e);
    batches = [];
  }

  const batchesById = indexBatchesByProductionId(batches);

  // 2) mintsDTO（肉付け）
  let mintsDTOById: Record<string, MintDTO> = {};
  try {
    mintsDTOById = (await fetchMintsByInspectionIdsHTTP(
      effectiveProductionIds,
    )) as any;
    const keys = Object.keys(mintsDTOById ?? {});
    log(
      "fetchMintsByInspectionIdsHTTP (dto) keys=",
      keys.length,
      "sampleKey=",
      keys[0],
    );
  } catch (e: any) {
    log("fetchMintsByInspectionIdsHTTP (dto) error=", e?.message ?? e);
    mintsDTOById = {};
  }

  // 3) 表示用 list row（tokenName / createdByName）
  let mintsListById: Record<string, MintListRowDTO> = {};
  try {
    mintsListById = await listMintsByInspectionIDsHTTP(effectiveProductionIds);
    const keys = Object.keys(mintsListById ?? {});
    log(
      "listMintsByInspectionIDsHTTP (list) keys=",
      keys.length,
      "sampleKey=",
      keys[0],
    );
  } catch (e: any) {
    log("listMintsByInspectionIDsHTTP (list) error=", e?.message ?? e);
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

  log("buildRowsJoined rows(length)=", rows.length, "sample[0]=", rows[0]);
  log("load end");
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

  const keys = Object.keys(m ?? {});
  log("loadMintsDTOMapByInspectionIds keys=", keys.length, "sampleKey=", keys[0]);

  return (m ?? {}) as Record<string, MintDTO>;
}

export async function loadMintDTOByInspectionId(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) return null;

  const m = await fetchMintByInspectionIdHTTP(iid);

  log("loadMintDTOByInspectionId iid=", iid, "result=", m ?? null);

  return (m ?? null) as MintDTO | null;
}
