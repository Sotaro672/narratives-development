// frontend/console/mintRequest/src/infrastructure/repository/mintRequestRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import type {
  InspectionBatchDTO,
  MintListRowDTO,
  MintDTO,
} from "../api/mintRequestApi";

// ✅ ここで DTO を定義して循環/参照エラーを避ける
export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  brandName?: string | null;

  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;

  // ✅ normalize で最終的に { type } に揃える（受け取りは Type / type 両対応）
  productIdTag?: { type?: string | null; Type?: string | null } | null;

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

// ★ NEW: /mint/inspections/{productionId} の detail DTO（バックエンド返却差異に強くするため緩め）
export type MintModelMetaEntryDTO = {
  modelNumber?: string | null;
  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;
};

export type MintRequestDetailDTO = {
  // id / productionId / inspectionId など揺れる可能性があるため任意
  productionId?: string | null;
  inspectionId?: string | null;

  // inspection batch（または同等）
  inspection?: InspectionBatchDTO | null;

  // mint（存在すれば）
  mint?: MintDTO | null;

  // product blueprint patch（存在すれば）
  productBlueprintPatch?: ProductBlueprintPatchDTO | null;

  // model variations -> modelMeta（存在すれば）
  modelMeta?: Record<string, MintModelMetaEntryDTO> | null;

  // 主要フィールド（detail の揺れ吸収用）
  tokenBlueprintId?: string | null;
  productName?: string | null;
  tokenName?: string | null;

  // その他バックエンド側が返すフィールドを落とさない
  [k: string]: any;
};

// ===============================
// ✅ /mint/requests response helpers
// ===============================

type MintRequestRowRaw = {
  id?: string | null;
  productionId?: string | null;
  inspectionId?: string | null;

  // “mint が埋め込まれて返る” 想定
  mint?: any | null;
  Mint?: any | null;

  // “list row 的に平坦化されて返る”可能性もある
  tokenName?: string | null;
  createdByName?: string | null;
  mintedAt?: string | null;
  minted?: boolean | null;

  [k: string]: any;
};

type MintRequestsPayloadRaw =
  | {
      rows?: MintRequestRowRaw[] | null;
      Rows?: MintRequestRowRaw[] | null;
      items?: MintRequestRowRaw[] | null;
      Items?: MintRequestRowRaw[] | null;
      data?: MintRequestRowRaw[] | null;
      Data?: MintRequestRowRaw[] | null;
      [k: string]: any;
    }
  | MintRequestRowRaw[];

// 🔙 BACKEND の BASE URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  return await user.getIdToken();
}

function buildHeaders(idToken: string): HeadersInit {
  return {
    Authorization: `Bearer ${idToken}`,
    "Content-Type": "application/json",
  };
}

// ---------------------------------------------------------
// helper: safe string
// ---------------------------------------------------------
function asTrimmedString(v: any): string {
  return typeof v === "string" ? v.trim() : String(v ?? "").trim();
}

function asMaybeString(v: any): string | null {
  const s = asTrimmedString(v);
  return s ? s : null;
}

// ---------------------------------------------------------
// helper: list row normalize（hook 側の “正” を前提に最小限）
// ---------------------------------------------------------
function normalizeMintListRow(v: any): MintListRowDTO {
  // ここは UI（hook/service）側で inspectionId を正として扱っているため、
  // 返却側のキーは “inspectionId として” 揃える（productionId/id 揺れは rowKey 側で吸収）
  const inspectionId = asMaybeString(v?.inspectionId ?? v?.productionId ?? v?.id) ?? null;

  const mintId = asMaybeString(v?.mintId ?? v?.id) ?? null;

  // ✅ tokenBlueprintId は lowerCamel を正として扱う（名揺れ吸収を削減）
  const tokenBlueprintId = asMaybeString(v?.tokenBlueprintId) ?? null;

  // ✅ tokenName も “tokenName” を正とする
  const tokenName = asMaybeString(v?.tokenName) ?? null;

  const createdByName = asMaybeString(v?.createdByName) ?? null;

  const mintedAt =
    typeof v?.mintedAt === "string" && v.mintedAt.trim() ? v.mintedAt.trim() : null;

  const minted = typeof v?.minted === "boolean" ? v.minted : Boolean(mintedAt);

  return {
    inspectionId,
    mintId,
    tokenBlueprintId,
    tokenName,
    createdByName,
    mintedAt,
    minted,
  } as any;
}

// ---------------------------------------------------------
// helper: MintDTO normalize（tokenBlueprint 周りの名揺れ吸収を削減）
// ---------------------------------------------------------
function normalizeMintDTO(v: any): MintDTO {
  const obj: any = { ...(v ?? {}) };

  // id
  obj.id = obj.id ?? "";

  // ✅ tokenBlueprintId / brandId は lowerCamel を正として扱う
  obj.brandId = obj.brandId ?? "";
  obj.tokenBlueprintId = obj.tokenBlueprintId ?? "";

  // inspectionId（ここは productionId と同一視される実装が残り得るため、最小限の互換は維持）
  obj.inspectionId = obj.inspectionId ?? obj.productionId ?? obj.ProductionID ?? "";

  obj.createdAt = obj.createdAt ?? null;
  obj.createdBy = obj.createdBy ?? "";
  obj.createdByName = obj.createdByName ?? null;

  // tokenName（あれば）
  obj.tokenName = obj.tokenName ?? null;

  obj.minted =
    typeof obj.minted === "boolean" ? obj.minted : Boolean(obj.mintedAt ?? null);
  obj.mintedAt = obj.mintedAt ?? null;

  obj.scheduledBurnDate = obj.scheduledBurnDate ?? null;
  obj.onChainTxSignature = obj.onChainTxSignature ?? null;

  return obj as MintDTO;
}

// ---------------------------------------------------------
// ✅ helper: ProductBlueprintPatch normalize（productIdTag を {type} に統一）
// ---------------------------------------------------------
function normalizeProductBlueprintPatch(v: any): ProductBlueprintPatchDTO | null {
  if (!v) return null;

  const rawTag = v?.productIdTag ?? v?.ProductIdTag ?? v?.product_id_tag ?? null;

  let tagType: string | null = null;

  if (rawTag) {
    tagType =
      asMaybeString(rawTag?.type) ??
      asMaybeString(rawTag?.Type) ??
      asMaybeString(rawTag?.TYPE);

    if (!tagType && typeof rawTag?.type === "object") {
      tagType =
        asMaybeString(rawTag?.type?.type) ??
        asMaybeString(rawTag?.type?.Type) ??
        null;
    }
    if (!tagType && typeof rawTag?.Type === "object") {
      tagType =
        asMaybeString(rawTag?.Type?.type) ??
        asMaybeString(rawTag?.Type?.Type) ??
        null;
    }

    if (!tagType && typeof rawTag === "string") {
      tagType = asMaybeString(rawTag);
    }
  }

  const out: ProductBlueprintPatchDTO = {
    productName: asMaybeString(v?.productName ?? v?.ProductName) ?? null,
    brandId: asMaybeString(v?.brandId ?? v?.BrandID ?? v?.BrandId) ?? null,
    brandName: asMaybeString(v?.brandName ?? v?.BrandName) ?? null,

    itemType: asMaybeString(v?.itemType ?? v?.ItemType) ?? null,
    fit: asMaybeString(v?.fit ?? v?.Fit) ?? null,
    material: asMaybeString(v?.material ?? v?.Material) ?? null,

    weight:
      typeof (v?.weight ?? v?.Weight) === "number"
        ? (v?.weight ?? v?.Weight)
        : Number(v?.weight ?? v?.Weight) || null,

    qualityAssurance:
      (v?.qualityAssurance ??
        v?.QualityAssurance ??
        v?.washTags ??
        v?.WashTags ??
        null) ?? null,

    productIdTag: tagType ? { type: tagType } : null,

    assigneeId:
      asMaybeString(v?.assigneeId ?? v?.AssigneeID ?? v?.AssigneeId) ?? null,
  };

  return out;
}

// ---------------------------------------------------------
// helper: productions -> productionIds（mint/inspections 用）
// ---------------------------------------------------------
function normalizeProductionIdFromProductionListItem(v: any): string {
  return String(
    v?.productionId ??
      v?.ProductionId ??
      v?.id ??
      v?.ID ??
      v?.production?.id ??
      v?.production?.ID ??
      v?.production?.productionId ??
      "",
  ).trim();
}

function normalizeProductBlueprintIdFromProductionListItem(v: any): string {
  return String(
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
}

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

// ===============================
// productions: productBlueprintId 解決（detail 用）
// ===============================

export async function fetchProductBlueprintIdByProductionIdHTTP(
  productionId: string,
): Promise<string | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) throw new Error("productionId が空です");

  const idToken = await getIdTokenOrThrow();

  const url1 = `${API_BASE}/productions/${encodeURIComponent(pid)}`;

  try {
    const res1 = await fetch(url1, { method: "GET", headers: buildHeaders(idToken) });

    if (res1.ok) {
      const j1 = (await res1.json()) as any;
      const pb1 = normalizeProductBlueprintIdFromProductionListItem(j1);
      return pb1 ? pb1 : null;
    }
  } catch (_e: any) {
    // noop
  }

  const url2 = `${API_BASE}/productions`;

  const res2 = await fetch(url2, { method: "GET", headers: buildHeaders(idToken) });

  if (!res2.ok) {
    throw new Error(`Failed to fetch productions: ${res2.status} ${res2.statusText}`);
  }

  const json2 = await res2.json();
  const items = normalizeProductionsPayload(json2);

  const hit =
    (items ?? []).find(
      (it: any) => normalizeProductionIdFromProductionListItem(it) === pid,
    ) ?? null;

  const pb2 = hit ? normalizeProductBlueprintIdFromProductionListItem(hit) : "";
  return pb2 ? pb2 : null;
}

async function fetchProductionIdsForCurrentCompanyHTTP(): Promise<string[]> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/productions`;

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  if (!res.ok) {
    throw new Error(`Failed to fetch productions: ${res.status} ${res.statusText}`);
  }

  const json = await res.json();
  const items = normalizeProductionsPayload(json);

  const ids: string[] = [];
  const seen = new Set<string>();
  for (const it of items) {
    const pid = normalizeProductionIdFromProductionListItem(it);
    if (!pid || seen.has(pid)) continue;
    seen.add(pid);
    ids.push(pid);
  }

  return ids;
}

// ===============================
// HTTP Repository (inspections)
// ===============================

function looksLikeInspectionBatchDTO(x: any): boolean {
  if (!x || typeof x !== "object") return false;
  return (
    Array.isArray(x.inspections) ||
    Array.isArray(x.Inspections) ||
    Array.isArray(x.results) ||
    Array.isArray(x.Results) ||
    Array.isArray(x.items) ||
    Array.isArray(x.Items)
  );
}

function normalizeMintRequestDetail(v: any): MintRequestDetailDTO | null {
  if (!v) return null;

  const pid =
    asMaybeString(v?.productionId ?? v?.ProductionID ?? v?.ProductionId ?? v?.id ?? v?.ID) ??
    null;

  const inspectionId =
    asMaybeString(
      v?.inspectionId ??
        v?.InspectionID ??
        v?.InspectionId ??
        v?.inspectionID ??
        v?.productionId ??
        v?.ProductionID ??
        v?.ProductionId,
    ) ?? null;

  // inspection 本体の取り出し（揺れ吸収）
  const inspectionRaw =
    v?.inspection ??
    v?.inspectionBatch ??
    v?.Inspection ??
    v?.InspectionBatch ??
    null;

  const looksLikeInspectionBatch =
    typeof v === "object" &&
    (Array.isArray((v as any)?.inspections) ||
      Array.isArray((v as any)?.Inspections) ||
      Array.isArray((v as any)?.results) ||
      Array.isArray((v as any)?.Results) ||
      Array.isArray((v as any)?.items) ||
      Array.isArray((v as any)?.Items));

  const inspection: InspectionBatchDTO | null =
    (inspectionRaw as any) ?? (looksLikeInspectionBatch ? (v as any) : null) ?? null;

  // mint 本体（揺れ吸収）
  const mintRaw = v?.mint ?? v?.Mint ?? v?.mintDTO ?? v?.MintDTO ?? null;
  const mint: MintDTO | null = mintRaw ? normalizeMintDTO(mintRaw) : null;

  // productBlueprintPatch（揺れ吸収）
  const pbpRaw =
    v?.productBlueprintPatch ??
    v?.productBlueprint ??
    v?.ProductBlueprintPatch ??
    v?.patch ??
    v?.Patch ??
    null;
  const productBlueprintPatch = normalizeProductBlueprintPatch(pbpRaw);

  // modelMeta（揺れ吸収）
  const modelMetaRaw =
    v?.modelMeta ?? v?.ModelMeta ?? v?.model_meta ?? v?.modelmeta ?? null;

  const modelMeta: Record<string, MintModelMetaEntryDTO> | null =
    modelMetaRaw && typeof modelMetaRaw === "object" ? modelMetaRaw : null;

  // ✅ detail の主要フィールド（UI 側で使うキー）
  // tokenBlueprintId は lowerCamel を正として扱う（名揺れ吸収を削減）
  const tokenBlueprintIdFromTop = asMaybeString(v?.tokenBlueprintId) ?? null;
  const tokenBlueprintIdFromMint = asMaybeString((mint as any)?.tokenBlueprintId) ?? null;
  const tokenBlueprintId = tokenBlueprintIdFromTop ?? tokenBlueprintIdFromMint ?? null;

  const productName =
    asMaybeString(v?.productName ?? v?.ProductName) ??
    asMaybeString((productBlueprintPatch as any)?.productName) ??
    null;

  const tokenName =
    asMaybeString(v?.tokenName) ??
    asMaybeString((mint as any)?.tokenName) ??
    null;

  return {
    ...(v ?? {}),
    productionId: pid,
    inspectionId,

    tokenBlueprintId,
    productName,
    tokenName,

    inspection: inspection ?? null,
    mint,
    productBlueprintPatch,
    modelMeta,
  };
}

export async function fetchMintRequestDetailByProductionIdHTTP(
  productionId: string,
): Promise<MintRequestDetailDTO | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) throw new Error("productionId が空です");

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(pid)}`;

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  if (res.status === 404) return null;

  if (!res.ok) {
    throw new Error(
      `Failed to fetch mint request detail: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as any;
  const out = normalizeMintRequestDetail(json);
  return out ?? null;
}

export async function fetchInspectionBatchesHTTP(): Promise<InspectionBatchDTO[]> {
  const productionIds = await fetchProductionIdsForCurrentCompanyHTTP();
  if (productionIds.length === 0) return [];
  return await fetchInspectionBatchesByProductionIdsHTTP(productionIds);
}

export async function fetchInspectionBatchesByProductionIdsHTTP(
  productionIds: string[],
): Promise<InspectionBatchDTO[]> {
  const ids = (productionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return [];

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/inspections?productionIds=${encodeURIComponent(
    ids.join(","),
  )}`;

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspections (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO[] | null | undefined;
  return json ?? [];
}

export async function fetchInspectionByProductionIdHTTP(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = String(productionId ?? "").trim();
  if (!trimmed) throw new Error("productionId が空です");

  // ✅ detail を優先（batch-shape のときだけ採用）
  try {
    const detail = await fetchMintRequestDetailByProductionIdHTTP(trimmed);
    const inspection = (detail?.inspection ?? null) as any;

    if (looksLikeInspectionBatchDTO(inspection)) {
      return inspection as InspectionBatchDTO;
    }
  } catch (_e: any) {
    // noop
  }

  // 🔙 fallback: list ルート
  const batches = await fetchInspectionBatchesByProductionIdsHTTP([trimmed]);
  const hit =
    batches.find(
      (b: any) =>
        String((b as any)?.productionId ?? (b as any)?.ProductionID ?? "").trim() === trimmed,
    ) ?? null;

  return hit ?? null;
}

// ===============================
// HTTP Repository (productBlueprint Patch)
// ===============================

export async function fetchProductBlueprintPatchHTTP(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/product_blueprints/${encodeURIComponent(
    productBlueprintId,
  )}/patch`;

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  if (res.status === 404) return null;

  if (!res.ok) {
    throw new Error(
      `Failed to fetch productBlueprintPatch: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as any;
  return normalizeProductBlueprintPatch(json) ?? null;
}

// ===============================
// HTTP Repository (brands for Mint)
// ===============================

type BrandRecordRaw = {
  id?: string;
  name?: string;
  ID?: string;
  Name?: string;
};

type BrandPageResultDTO = {
  items?: BrandRecordRaw[];
  Items?: BrandRecordRaw[];
};

export async function fetchBrandsForMintHTTP(): Promise<BrandForMintDTO[]> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/brands`;

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch brands (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as BrandPageResultDTO | null | undefined;

  const rawItems: BrandRecordRaw[] = json?.items ?? json?.Items ?? [];

  return rawItems
    .map((b) => ({
      id: (b.id ?? b.ID ?? "").trim(),
      name: (b.name ?? b.Name ?? "").trim(),
    }))
    .filter((b) => b.id && b.name);
}

// ===============================
// HTTP Repository (tokenBlueprints for Mint)
// ===============================

type TokenBlueprintRecordRaw = {
  id?: string;
  name?: string;
  symbol?: string;
  iconUrl?: string;

  ID?: string;
  Name?: string;
  Symbol?: string;
  IconUrl?: string;
};

type TokenBlueprintPageResultDTO = {
  items?: TokenBlueprintRecordRaw[];
  Items?: TokenBlueprintRecordRaw[];
};

export async function fetchTokenBlueprintsByBrandHTTP(
  brandId: string,
): Promise<TokenBlueprintForMintDTO[]> {
  const trimmed = brandId.trim();
  if (!trimmed) return [];

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/token_blueprints?brandId=${encodeURIComponent(
    trimmed,
  )}`;

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  if (res.status === 404) return [];

  if (!res.ok) {
    throw new Error(
      `Failed to fetch tokenBlueprints (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as
    | TokenBlueprintPageResultDTO
    | TokenBlueprintRecordRaw[]
    | null
    | undefined;

  const rawItems: TokenBlueprintRecordRaw[] = Array.isArray(json)
    ? json
    : json?.items ?? json?.Items ?? [];

  return rawItems
    .map((tb) => ({
      id: (tb.id ?? tb.ID ?? "").trim(),
      name: (tb.name ?? tb.Name ?? "").trim(),
      symbol: (tb.symbol ?? tb.Symbol ?? "").trim(),
      iconUrl: (tb.iconUrl ?? tb.IconUrl ?? "").trim() || undefined,
    }))
    .filter((tb) => tb.id && tb.name && tb.symbol);
}

// ===============================
// HTTP Repository (model variations for Mint)
// ===============================

export type ModelVariationForMintDTO = {
  id: string;
  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;
};

function normalizeModelVariationForMintDTO(v: any): ModelVariationForMintDTO | null {
  if (!v) return null;

  const id = String(v?.id ?? v?.ID ?? "").trim();
  if (!id) return null;

  const modelNumber =
    String(v?.modelNumber ?? v?.ModelNumber ?? "").trim() || null;
  const size = String(v?.size ?? v?.Size ?? "").trim() || null;

  const colorObj = v?.color ?? v?.Color ?? null;

  const colorName =
    String(
      v?.colorName ??
        v?.ColorName ??
        colorObj?.name ??
        colorObj?.Name ??
        "",
    ).trim() || null;

  const rgbRaw = v?.rgb ?? v?.RGB ?? colorObj?.rgb ?? colorObj?.RGB ?? null;

  const rgb =
    typeof rgbRaw === "number"
      ? rgbRaw
      : Number.isFinite(Number(rgbRaw))
        ? Number(rgbRaw)
        : null;

  return { id, modelNumber, size, colorName, rgb };
}

export async function fetchModelVariationByIdForMintHTTP(
  variationId: string,
): Promise<ModelVariationForMintDTO | null> {
  const vid = String(variationId ?? "").trim();
  if (!vid) throw new Error("variationId が空です");

  const idToken = await getIdTokenOrThrow();

  const candidates = [
    `${API_BASE}/models/variations/${encodeURIComponent(vid)}`,
    `${API_BASE}/model/variations/${encodeURIComponent(vid)}`,
  ];

  for (const url of candidates) {
    try {
      const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

      if (res.status === 404 || res.status === 405) continue;
      if (res.status >= 500) continue;

      if (!res.ok) {
        throw new Error(
          `Failed to fetch model variation: ${res.status} ${res.statusText}`,
        );
      }

      const json = (await res.json()) as any;
      return normalizeModelVariationForMintDTO(json);
    } catch (_e: any) {
      continue;
    }
  }

  return null;
}

// ===============================
// HTTP Repository (mints via /mint/requests only)
// ===============================

function normalizeMintRequestsRows(json: any): MintRequestRowRaw[] {
  if (!json) return [];
  if (Array.isArray(json)) return json as MintRequestRowRaw[];

  const rows =
    (json as any)?.rows ??
    (json as any)?.Rows ??
    (json as any)?.items ??
    (json as any)?.Items ??
    (json as any)?.data ??
    (json as any)?.Data ??
    null;

  return Array.isArray(rows) ? (rows as MintRequestRowRaw[]) : [];
}

function extractRowKeyAsProductionId(row: any): string {
  return String(
    row?.productionId ??
      row?.ProductionID ??
      row?.ProductionId ??
      row?.inspectionId ??
      row?.InspectionID ??
      row?.InspectionId ??
      row?.id ??
      row?.ID ??
      "",
  ).trim();
}

async function fetchMintRequestsRowsRaw(
  ids: string[],
  view: "management" | "dto" | "list" | null,
): Promise<MintRequestRowRaw[]> {
  const idToken = await getIdTokenOrThrow();

  const base = `${API_BASE}/mint/requests?productionIds=${encodeURIComponent(
    ids.join(","),
  )}`;
  const url = view ? `${base}&view=${encodeURIComponent(view)}` : base;

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  if (res.status === 404) return [];
  if (!res.ok) {
    throw new Error(
      `Failed to fetch mint requests: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as MintRequestsPayloadRaw | null | undefined;
  return normalizeMintRequestsRows(json);
}

export async function fetchMintByInspectionIdHTTP(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) throw new Error("inspectionId が空です");

  try {
    const rows = await fetchMintRequestsRowsRaw([iid], "management");
    const row =
      (rows ?? []).find((r) => extractRowKeyAsProductionId(r) === iid) ??
      rows?.[0] ??
      null;
    if (!row) return null;

    const mintRaw = row?.mint ?? row?.Mint ?? null;
    if (mintRaw) return normalizeMintDTO(mintRaw);

    return normalizeMintDTO(row);
  } catch (_e: any) {
    return null;
  }
}

export async function fetchMintListRowsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  try {
    const rows = await fetchMintRequestsRowsRaw(ids, "management");

    const out: Record<string, MintListRowDTO> = {};
    for (const r of rows ?? []) {
      const key = extractRowKeyAsProductionId(r);
      if (!key) continue;

      const base =
        (r?.mint ?? r?.Mint ?? null) ? (r?.mint ?? r?.Mint) : (r as any);

      const merged = {
        ...(base ?? {}),
        inspectionId: key,
        productionId: key,
        tokenName: (r as any)?.tokenName ?? (base as any)?.tokenName ?? null,
        createdByName:
          (r as any)?.createdByName ?? (base as any)?.createdByName ?? null,
        mintedAt: (r as any)?.mintedAt ?? (base as any)?.mintedAt ?? null,
        minted:
          typeof (r as any)?.minted === "boolean"
            ? (r as any).minted
            : (base as any)?.minted,
      };

      out[key] = normalizeMintListRow(merged);
    }

    return out;
  } catch (_e: any) {
    return {};
  }
}

export async function fetchMintsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  try {
    const rows = await fetchMintRequestsRowsRaw(ids, "management");

    const out: Record<string, MintDTO> = {};
    for (const r of rows ?? []) {
      const key = extractRowKeyAsProductionId(r);
      if (!key) continue;

      const mintRaw = r?.mint ?? r?.Mint ?? null;
      if (mintRaw) {
        out[key] = normalizeMintDTO(mintRaw);
        continue;
      }
      out[key] = normalizeMintDTO(r);
    }

    return out;
  } catch (_e: any) {
    return {};
  }
}

export async function listMintsByInspectionIDsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  return await fetchMintListRowsByInspectionIdsHTTP(inspectionIds);
}

// ===============================
// HTTP Repository (mint request)
// ===============================

export async function postMintRequestHTTP(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) throw new Error("productionId が空です");

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(trimmed)}/request`;

  const payload: {
    tokenBlueprintId: string;
    scheduledBurnDate?: string;
  } = {
    tokenBlueprintId: tokenBlueprintId.trim(),
  };

  if (scheduledBurnDate && scheduledBurnDate.trim()) {
    payload.scheduledBurnDate = scheduledBurnDate.trim();
  }

  const res = await fetch(url, {
    method: "POST",
    headers: buildHeaders(idToken),
    body: JSON.stringify(payload),
  });

  if (res.status === 404) return null;

  if (!res.ok) {
    throw new Error(
      `Failed to post mint request: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO | null | undefined;
  return json ?? null;
}
