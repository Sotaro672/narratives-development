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
// helper: list row normalize（バックエンド返却差異に強くする）
// ---------------------------------------------------------
function normalizeMintListRow(v: any): MintListRowDTO {
  const inspectionId =
    String(
      v?.inspectionId ??
        v?.InspectionID ??
        v?.inspectionID ??
        v?.productionId ??
        v?.ProductionID ??
        "",
    ).trim() || null;

  const mintId =
    String(v?.mintId ?? v?.MintID ?? v?.id ?? v?.ID ?? "").trim() || null;

  const tokenBlueprintId =
    String(
      v?.tokenBlueprintId ??
        v?.TokenBlueprintID ??
        v?.tokenBlueprint ??
        v?.TokenBlueprint ??
        "",
    ).trim() || null;

  const tokenName =
    String(
      v?.tokenName ??
        v?.tokenBlueprintName ??
        v?.name ??
        tokenBlueprintId ??
        "",
    ).trim() || null;

  const createdByName =
    String(v?.createdByName ?? v?.CreatedByName ?? v?.createdBy ?? "").trim() ||
    null;

  const mintedAtRaw = v?.mintedAt ?? v?.MintedAt ?? null;
  const mintedAt =
    typeof mintedAtRaw === "string" && mintedAtRaw.trim()
      ? mintedAtRaw.trim()
      : null;

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
// helper: MintDTO normalize（バックエンド返却差異に強くする）
// ---------------------------------------------------------
function normalizeMintDTO(v: any): MintDTO {
  const obj: any = { ...(v ?? {}) };

  obj.id = obj.id ?? obj.ID ?? "";
  obj.brandId = obj.brandId ?? obj.BrandID ?? "";
  obj.tokenBlueprintId = obj.tokenBlueprintId ?? obj.TokenBlueprintID ?? "";
  obj.inspectionId =
    obj.inspectionId ??
    obj.InspectionID ??
    obj.inspectionID ??
    obj.productionId ??
    obj.ProductionID ??
    "";

  obj.createdAt = obj.createdAt ?? obj.CreatedAt ?? null;
  obj.createdBy = obj.createdBy ?? obj.CreatedBy ?? "";
  obj.createdByName = obj.createdByName ?? obj.CreatedByName ?? null;

  obj.minted =
    typeof obj.minted === "boolean" ? obj.minted : Boolean(obj.mintedAt);
  obj.mintedAt = obj.mintedAt ?? obj.MintedAt ?? null;

  obj.scheduledBurnDate =
    obj.scheduledBurnDate ?? obj.ScheduledBurnDate ?? null;

  obj.onChainTxSignature =
    obj.onChainTxSignature ?? obj.OnChainTxSignature ?? null;

  return obj as MintDTO;
}

// ---------------------------------------------------------
// helper: productions -> productionIds（mint/inspections 用）
// ---------------------------------------------------------
function normalizeProductionIdFromProductionListItem(v: any): string {
  return String(
    v?.productionId ??
      v?.id ??
      v?.ID ??
      v?.production?.id ??
      v?.production?.ID ??
      v?.production?.productionId ??
      "",
  ).trim();
}

// ★更新: /productions から productBlueprintId を拾う（返却差異を吸収）
function normalizeProductBlueprintIdFromProductionListItem(v: any): string {
  const raw =
    v?.productBlueprintId ??
    v?.productBlueprintID ??
    v?.ProductBlueprintId ??
    v?.ProductBlueprintID ?? // ✅ これが今回のキー
    v?.production?.productBlueprintId ??
    v?.production?.productBlueprintID ??
    v?.production?.ProductBlueprintId ??
    v?.production?.ProductBlueprintID ??
    "";

  return String(raw ?? "").trim();
}

// ★追加: /productions の返却が配列/ページングどちらでも吸収
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

/**
 * ✅ productBlueprintId を productionId から解決する
 * - まず /productions/{id} を試す（存在すれば一発）
 * - なければ /productions 一覧から検索
 */
export async function fetchProductBlueprintIdByProductionIdHTTP(
  productionId: string,
): Promise<string | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) throw new Error("productionId が空です");

  const idToken = await getIdTokenOrThrow();

  // 1) /productions/{id} を試す（存在する環境なら最短）
  const url1 = `${API_BASE}/productions/${encodeURIComponent(pid)}`;

  try {
    const res1 = await fetch(url1, {
      method: "GET",
      headers: buildHeaders(idToken),
    });

    if (res1.ok) {
      const j1 = (await res1.json()) as any;

      // ✅ 重要：ProductBlueprintID などの揺れも吸収して拾う
      const pb1 = normalizeProductBlueprintIdFromProductionListItem(j1);

      return pb1 ? pb1 : null;
    }
  } catch {
    // ignore -> fallback list
  }

  // 2) /productions 一覧から探す
  const url2 = `${API_BASE}/productions`;

  const res2 = await fetch(url2, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  if (!res2.ok) {
    throw new Error(
      `Failed to fetch productions: ${res2.status} ${res2.statusText}`,
    );
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

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch productions: ${res.status} ${res.statusText}`,
    );
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

export async function fetchInspectionBatchesHTTP(): Promise<InspectionBatchDTO[]> {
  const productionIds = await fetchProductionIdsForCurrentCompanyHTTP();

  if (productionIds.length === 0) {
    return [];
  }

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

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspections (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO[] | null | undefined;
  const out = json ?? [];
  return out;
}

export async function fetchInspectionByProductionIdHTTP(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) {
    throw new Error("productionId が空です");
  }

  const batches = await fetchInspectionBatchesByProductionIdsHTTP([trimmed]);
  const hit =
    batches.find(
      (b: any) => String((b as any)?.productionId ?? "").trim() === trimmed,
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

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  if (res.status === 404) {
    return null;
  }

  if (!res.ok) {
    throw new Error(
      `Failed to fetch productBlueprintPatch: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as ProductBlueprintPatchDTO | null | undefined;
  return json ?? null;
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

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch brands (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as BrandPageResultDTO | null | undefined;

  const rawItems: BrandRecordRaw[] = json?.items ?? json?.Items ?? [];

  const mapped: BrandForMintDTO[] = rawItems
    .map((b) => ({
      id: (b.id ?? b.ID ?? "").trim(),
      name: (b.name ?? b.Name ?? "").trim(),
    }))
    .filter((b) => b.id && b.name);

  return mapped;
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
  if (!trimmed) {
    return [];
  }

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/token_blueprints?brandId=${encodeURIComponent(
    trimmed,
  )}`;

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  if (res.status === 404) {
    return [];
  }

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

  const mapped: TokenBlueprintForMintDTO[] = rawItems
    .map((tb) => ({
      id: (tb.id ?? tb.ID ?? "").trim(),
      name: (tb.name ?? tb.Name ?? "").trim(),
      symbol: (tb.symbol ?? tb.Symbol ?? "").trim(),
      iconUrl: (tb.iconUrl ?? tb.IconUrl ?? "").trim() || undefined,
    }))
    .filter((tb) => tb.id && tb.name && tb.symbol);

  return mapped;
}

// ===============================
// HTTP Repository (mints)
// ===============================

async function fetchMintsMapRaw(
  ids: string[],
  view: "list" | "dto" | null,
): Promise<Record<string, any>> {
  const idToken = await getIdTokenOrThrow();

  const base = `${API_BASE}/mint/mints?inspectionIds=${encodeURIComponent(
    ids.join(","),
  )}`;
  const url = view ? `${base}&view=${encodeURIComponent(view)}` : base;

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  if (res.status === 404) return {};
  if (!res.ok) {
    throw new Error(`Failed to fetch mints: ${res.status} ${res.statusText}`);
  }

  const json = (await res.json()) as Record<string, any> | null | undefined;
  const raw = json ?? {};
  return raw;
}

export async function fetchMintByInspectionIdHTTP(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) throw new Error("inspectionId が空です");

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/mints/${encodeURIComponent(iid)}`;

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  if (res.status === 404) return null;

  if (!res.ok) {
    throw new Error(
      `Failed to fetch mint by inspectionId: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as any;
  if (!json) return null;

  const out = normalizeMintDTO(json);
  return out;
}

async function fetchMintListRowsByInspectionIdsFallback(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter(Boolean);

  if (ids.length === 0) return {};

  const settled = await Promise.all(
    ids.map(async (inspectionId) => {
      try {
        const m = await fetchMintByInspectionIdHTTP(inspectionId);
        return { inspectionId, mint: m };
      } catch {
        return { inspectionId, mint: null };
      }
    }),
  );

  const out: Record<string, MintListRowDTO> = {};
  for (const it of settled) {
    if (!it.mint) continue;

    const v = {
      ...(it.mint as any),
      inspectionId: it.inspectionId,
      mintId: (it.mint as any).id ?? null,
      tokenBlueprintId: (it.mint as any).tokenBlueprintId ?? null,
      createdByName:
        (it.mint as any).createdByName ?? (it.mint as any).createdBy ?? null,
      mintedAt: (it.mint as any).mintedAt ?? null,
    };

    out[it.inspectionId] = normalizeMintListRow(v);
  }

  return out;
}

export async function fetchMintListRowsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  try {
    let raw: Record<string, any> = {};
    try {
      raw = await fetchMintsMapRaw(ids, "list");
    } catch {
      raw = await fetchMintsMapRaw(ids, null);
    }

    const out: Record<string, MintListRowDTO> = {};
    for (const [k, v] of Object.entries(raw ?? {})) {
      const key = String(k ?? "").trim();
      if (!key) continue;
      out[key] = normalizeMintListRow(v);
    }

    return out;
  } catch {
    return await fetchMintListRowsByInspectionIdsFallback(ids);
  }
}

export async function fetchMintsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  let raw: Record<string, any> = {};
  try {
    raw = await fetchMintsMapRaw(ids, "dto");
  } catch {
    raw = await fetchMintsMapRaw(ids, null);
  }

  const out: Record<string, MintDTO> = {};
  for (const [k, v] of Object.entries(raw ?? {})) {
    const key = String(k ?? "").trim();
    if (!key) continue;
    out[key] = normalizeMintDTO(v);
  }

  return out;
}

export async function listMintsByInspectionIDsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const m = await fetchMintListRowsByInspectionIdsHTTP(inspectionIds);
  return m;
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
  if (!trimmed) {
    throw new Error("productionId が空です");
  }

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
