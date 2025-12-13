// frontend/console/mintRequest/src/infrastructure/repository/mintRequestRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import type {
  InspectionBatchDTO,
  MintListRowDTO,
  MintDTO,
} from "../api/mintRequestApi";

// ✅ ここで DTO を定義して循環/参照エラーを避ける
// （application/mintRequestService 側の export 変更に依存しない）
export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  brandName?: string | null; // MintHandler が付与する

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

const LOG_PREFIX = "[mintRequest/mintRequestRepositoryHTTP]";

function log(...args: any[]) {
  // eslint-disable-next-line no-console
  console.log(LOG_PREFIX, ...args);
}

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

  // ✅ Products は画面に渡さない方針ならここで触らない
  // obj.products = obj.products ?? obj.Products ?? [];

  obj.createdAt = obj.createdAt ?? obj.CreatedAt ?? null;
  obj.createdBy = obj.createdBy ?? obj.CreatedBy ?? "";
  obj.createdByName = obj.createdByName ?? obj.CreatedByName ?? null;

  obj.minted =
    typeof obj.minted === "boolean" ? obj.minted : Boolean(obj.mintedAt);
  obj.mintedAt = obj.mintedAt ?? obj.MintedAt ?? null;

  obj.scheduledBurnDate = obj.scheduledBurnDate ?? obj.ScheduledBurnDate ?? null;

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
      v?.production?.id ??
      v?.production?.productionId ??
      "",
  ).trim();
}

async function fetchProductionIdsForCurrentCompanyHTTP(): Promise<string[]> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/productions`;
  log("fetchProductionIdsForCurrentCompanyHTTP url=", url);

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log(
    "fetchProductionIdsForCurrentCompanyHTTP status=",
    res.status,
    res.statusText,
  );

  if (!res.ok) {
    throw new Error(
      `Failed to fetch productions: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as any[] | null | undefined;
  const items = json ?? [];

  const ids: string[] = [];
  const seen = new Set<string>();
  for (const it of items) {
    const pid = normalizeProductionIdFromProductionListItem(it);
    if (!pid || seen.has(pid)) continue;
    seen.add(pid);
    ids.push(pid);
  }

  log(
    "fetchProductionIdsForCurrentCompanyHTTP result len=",
    ids.length,
    "sample[0..4]=",
    ids.slice(0, 5),
  );

  return ids;
}

// ===============================
// HTTP Repository (inspections)
// ===============================

/**
 * ✅ New flow: /productions で productionIds を作り、
 * /mint/inspections?productionIds=a,b,c を叩く。
 */
export async function fetchInspectionBatchesHTTP(): Promise<InspectionBatchDTO[]> {
  const productionIds = await fetchProductionIdsForCurrentCompanyHTTP();

  if (productionIds.length === 0) {
    log("fetchInspectionBatchesHTTP productionIds is empty -> return []");
    return [];
  }

  return await fetchInspectionBatchesByProductionIdsHTTP(productionIds);
}

/**
 * ✅ 直接 productionIds を指定して mint/inspections を叩く版
 * GET /mint/inspections?productionIds=a,b,c
 */
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
  log(
    "fetchInspectionBatchesByProductionIdsHTTP url=",
    url,
    "ids.length=",
    ids.length,
    "sample[0..4]=",
    ids.slice(0, 5),
  );

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log(
    "fetchInspectionBatchesByProductionIdsHTTP status=",
    res.status,
    res.statusText,
  );

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspections (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO[] | null | undefined;
  const out = json ?? [];
  log(
    "fetchInspectionBatchesByProductionIdsHTTP result length=",
    out.length,
    "sample[0]=",
    out[0],
  );
  return out;
}

export async function fetchInspectionByProductionIdHTTP(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) {
    throw new Error("productionId が空です");
  }

  // ✅ 新エンドポイント経由で 1 件だけ取りたい場合は productionIds を 1 個にして叩く
  const batches = await fetchInspectionBatchesByProductionIdsHTTP([trimmed]);
  const hit = batches.find((b: any) => String((b as any)?.productionId ?? "").trim() === trimmed) ?? null;

  log("fetchInspectionByProductionIdHTTP productionId=", trimmed, "hit=", hit);
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
  log("fetchProductBlueprintPatchHTTP url=", url);

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log("fetchProductBlueprintPatchHTTP status=", res.status, res.statusText);

  if (res.status === 404) {
    return null;
  }

  if (!res.ok) {
    throw new Error(
      `Failed to fetch productBlueprintPatch: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as ProductBlueprintPatchDTO | null | undefined;
  log("fetchProductBlueprintPatchHTTP result=", json);
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
  log("fetchBrandsForMintHTTP url=", url);

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log("fetchBrandsForMintHTTP status=", res.status, res.statusText);

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

  log(
    "fetchBrandsForMintHTTP result length=",
    mapped.length,
    "sample[0]=",
    mapped[0],
  );
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
  log("fetchTokenBlueprintsByBrandHTTP url=", url);

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log("fetchTokenBlueprintsByBrandHTTP status=", res.status, res.statusText);

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

  log(
    "fetchTokenBlueprintsByBrandHTTP result length=",
    mapped.length,
    "sample[0]=",
    mapped[0],
  );
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

  log("fetchMintsMapRaw url=", url, "ids.length=", ids.length, "view=", view);

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log("fetchMintsMapRaw status=", res.status, res.statusText, "url=", url);

  if (res.status === 404) return {};
  if (!res.ok) {
    throw new Error(`Failed to fetch mints: ${res.status} ${res.statusText}`);
  }

  const json = (await res.json()) as Record<string, any> | null | undefined;
  const raw = json ?? {};
  const keys = Object.keys(raw);
  log(
    "fetchMintsMapRaw response keys=",
    keys.length,
    "sampleKey=",
    keys[0],
    "sampleVal=",
    keys[0] ? raw[keys[0]] : undefined,
  );
  return raw;
}

/**
 * ✅ 単発: inspectionId(=productionId) で 1 件取得
 * backend: GET /mint/mints/{inspectionId}
 */
export async function fetchMintByInspectionIdHTTP(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) throw new Error("inspectionId が空です");

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/mints/${encodeURIComponent(iid)}`;
  log("fetchMintByInspectionIdHTTP url=", url);

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log("fetchMintByInspectionIdHTTP status=", res.status, res.statusText);

  if (res.status === 404) return null;

  if (!res.ok) {
    throw new Error(
      `Failed to fetch mint by inspectionId: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as any;
  log("fetchMintByInspectionIdHTTP raw=", json);
  if (!json) return null;

  const out = normalizeMintDTO(json);
  log("fetchMintByInspectionIdHTTP normalized=", out);
  return out;
}

/**
 * ✅ フォールバック: inspectionIds を 1件取得で回収して list row を組み立てる
 */
async function fetchMintListRowsByInspectionIdsFallback(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter(Boolean);

  if (ids.length === 0) return {};

  log(
    "fetchMintListRowsByInspectionIdsFallback start ids.length=",
    ids.length,
    "sample[0..4]=",
    ids.slice(0, 5),
  );

  const settled = await Promise.all(
    ids.map(async (inspectionId) => {
      try {
        const m = await fetchMintByInspectionIdHTTP(inspectionId);
        return { inspectionId, mint: m };
      } catch (e: any) {
        log(
          "fetchMintListRowsByInspectionIdsFallback error inspectionId=",
          inspectionId,
          e?.message ?? e,
        );
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
      createdByName: (it.mint as any).createdByName ?? (it.mint as any).createdBy ?? null,
      mintedAt: (it.mint as any).mintedAt ?? null,
    };

    out[it.inspectionId] = normalizeMintListRow(v);
  }

  const keys = Object.keys(out);
  log(
    "fetchMintListRowsByInspectionIdsFallback end keys=",
    keys.length,
    "sampleKey=",
    keys[0],
    "sampleVal=",
    keys[0] ? out[keys[0]] : undefined,
  );

  return out;
}

/**
 * ✅ 一覧用: inspectionIds をまとめて渡して、mints(list row) を取得する。
 */
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
    } catch (e: any) {
      log(
        "fetchMintListRowsByInspectionIdsHTTP fallback to no-view because:",
        e?.message ?? e,
      );
      raw = await fetchMintsMapRaw(ids, null);
    }

    const out: Record<string, MintListRowDTO> = {};
    for (const [k, v] of Object.entries(raw ?? {})) {
      const key = String(k ?? "").trim();
      if (!key) continue;
      out[key] = normalizeMintListRow(v);
    }

    const keys = Object.keys(out);
    log(
      "fetchMintListRowsByInspectionIdsHTTP normalized keys=",
      keys.length,
      "sampleKey=",
      keys[0],
      "sampleVal=",
      keys[0] ? out[keys[0]] : undefined,
    );
    return out;
  } catch (e: any) {
    log(
      "fetchMintListRowsByInspectionIdsHTTP fallback to per-id fetch because:",
      e?.message ?? e,
    );
    return await fetchMintListRowsByInspectionIdsFallback(ids);
  }
}

/**
 * ✅ 詳細DTO用: inspectionIds をまとめて渡して、mints(MintDTO) を取得する。
 */
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
  } catch (e: any) {
    log(
      "fetchMintsByInspectionIdsHTTP fallback to no-view because:",
      e?.message ?? e,
    );
    raw = await fetchMintsMapRaw(ids, null);
  }

  const out: Record<string, MintDTO> = {};
  for (const [k, v] of Object.entries(raw ?? {})) {
    const key = String(k ?? "").trim();
    if (!key) continue;
    out[key] = normalizeMintDTO(v);
  }

  const keys = Object.keys(out);
  log(
    "fetchMintsByInspectionIdsHTTP normalized keys=",
    keys.length,
    "sampleKey=",
    keys[0],
    "sampleVal=",
    keys[0] ? out[keys[0]] : undefined,
  );
  return out;
}

/**
 * ✅ 追加: 画面（service/hook）からはこちらを呼ぶ想定のエイリアス
 */
export async function listMintsByInspectionIDsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  log(
    "listMintsByInspectionIDsHTTP called ids=",
    (inspectionIds ?? []).slice(0, 10),
    "len=",
    (inspectionIds ?? []).length,
  );
  const m = await fetchMintListRowsByInspectionIdsHTTP(inspectionIds);
  log("listMintsByInspectionIDsHTTP done keys=", Object.keys(m ?? {}).length);
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

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(
    trimmed,
  )}/request`;

  const payload: {
    tokenBlueprintId: string;
    scheduledBurnDate?: string;
  } = {
    tokenBlueprintId: tokenBlueprintId.trim(),
  };

  if (scheduledBurnDate && scheduledBurnDate.trim()) {
    payload.scheduledBurnDate = scheduledBurnDate.trim();
  }

  log("postMintRequestHTTP url=", url, "payload=", payload);

  const res = await fetch(url, {
    method: "POST",
    headers: buildHeaders(idToken),
    body: JSON.stringify(payload),
  });

  log("postMintRequestHTTP status=", res.status, res.statusText);

  if (res.status === 404) return null;

  if (!res.ok) {
    throw new Error(
      `Failed to post mint request: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO | null | undefined;
  log("postMintRequestHTTP result=", json);
  return json ?? null;
}
