// frontend/console/mintRequest/src/infrastructure/repository/mintRequestRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import type {
  InspectionBatchDTO,
  MintListRowDTO,
  MintDTO,
} from "../api/mintRequestApi";
import type {
  ProductBlueprintPatchDTO,
  BrandForMintDTO,
  TokenBlueprintForMintDTO,
} from "../../application/mintRequestService";

// 🔙 BACKEND の BASE URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as
    | string
    | undefined)?.replace(/\/+$/g, "") ?? "";

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
  // ✅ 新DTO（backend）想定:
  // {
  //   inspectionId, mintId, tokenBlueprintId,
  //   tokenName, createdByName, mintedAt (RFC3339 | null)
  // }
  //
  // ✅ 旧DTO / 互換フォールバック:
  // { tokenName, createdByName, mintedAt } や MintDTO に近い形を吸収

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

  // mintedAt は RFC3339 でも yyyy/mm/dd でも「stringなら通す」
  const mintedAtRaw = v?.mintedAt ?? v?.MintedAt ?? null;
  const mintedAt =
    typeof mintedAtRaw === "string" && mintedAtRaw.trim()
      ? mintedAtRaw.trim()
      : null;

  // minted が無い場合は mintedAt で推定（一覧のステータス判定用）
  const minted =
    typeof v?.minted === "boolean" ? v.minted : Boolean(mintedAt);

  return {
    // フロント側 MintListRowDTO の定義に “inspectionId 等” が無い場合でも、
    // as any で保持しておくとデバッグに役立つ（必要なら型定義も更新してください）
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

  // camel / Pascal / 別名フォールバック
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

  // ✅ 画面に Products を渡さない方針なので削除
  // obj.products = obj.products ?? obj.Products ?? [];

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

// ===============================
// HTTP Repository (inspections)
// ===============================

export async function fetchInspectionBatchesHTTP(): Promise<InspectionBatchDTO[]> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/inspections`;
  log("fetchInspectionBatchesHTTP url=", url);

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log("fetchInspectionBatchesHTTP status=", res.status, res.statusText);

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspections (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO[] | null | undefined;
  const out = json ?? [];
  log(
    "fetchInspectionBatchesHTTP result length=",
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

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/products/inspections?productionId=${encodeURIComponent(
    trimmed,
  )}`;
  log("fetchInspectionByProductionIdHTTP url=", url);

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log("fetchInspectionByProductionIdHTTP status=", res.status, res.statusText);

  if (res.status === 404) {
    return null;
  }

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspection by productionId: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO | null | undefined;
  log("fetchInspectionByProductionIdHTTP result=", json);
  return json ?? null;
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
    // 呼び出し元で fallback できるように throw
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
    raw[keys[0]],
  );
  return raw;
}

/**
 * ✅ 単発: mintId で 1 件取得
 * backend: GET /mint/mints/{mintId}
 *
 * NOTE:
 * - 詳細用の MintDTO を返す前提
 */
export async function fetchMintByMintIdHTTP(
  mintId: string,
): Promise<MintDTO | null> {
  const mid = String(mintId ?? "").trim();
  if (!mid) {
    throw new Error("mintId が空です");
  }

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/mints/${encodeURIComponent(mid)}`;
  log("fetchMintByMintIdHTTP url=", url);

  const res = await fetch(url, {
    method: "GET",
    headers: buildHeaders(idToken),
  });

  log("fetchMintByMintIdHTTP status=", res.status, res.statusText);

  if (res.status === 404) return null;

  if (!res.ok) {
    throw new Error(
      `Failed to fetch mint by mintId: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as any;
  log("fetchMintByMintIdHTTP raw=", json);
  if (!json) return null;

  const out = normalizeMintDTO(json);
  log("fetchMintByMintIdHTTP normalized=", out);
  return out;
}

/**
 * ✅ フォールバック: mintIds を 1件取得で回収して一覧行 DTO を組み立てる
 *
 * - /mint/mints?inspectionIds=... が 500 でも画面が成立するようにする
 * - 戻り map の key は mintId
 */
async function fetchMintListRowsByMintIdsFallback(
  mintIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (mintIds ?? []).map((s) => String(s ?? "").trim()).filter(Boolean);
  if (ids.length === 0) return {};

  log(
    "fetchMintListRowsByMintIdsFallback start ids.length=",
    ids.length,
    "sample[0..4]=",
    ids.slice(0, 5),
  );

  const settled = await Promise.all(
    ids.map(async (mintId) => {
      try {
        const m = await fetchMintByMintIdHTTP(mintId);
        return { mintId, mint: m };
      } catch (e: any) {
        log(
          "fetchMintListRowsByMintIdsFallback error mintId=",
          mintId,
          e?.message ?? e,
        );
        return { mintId, mint: null };
      }
    }),
  );

  const out: Record<string, MintListRowDTO> = {};
  for (const it of settled) {
    if (!it.mint) continue;

    // normalizeMintListRow が吸収できる形に寄せる
    const v = {
      ...(it.mint as any),
      mintId: (it.mint as any).id || it.mintId,
      inspectionId: (it.mint as any).inspectionId || null,
    };

    out[it.mintId] = normalizeMintListRow(v);
  }

  const keys = Object.keys(out);
  log(
    "fetchMintListRowsByMintIdsFallback end keys=",
    keys.length,
    "sampleKey=",
    keys[0],
    "sampleVal=",
    out[keys[0]],
  );

  return out;
}

/**
 * ✅ 一覧用: ids をまとめて渡して、mints(list row) を取得する。
 *
 * まずは従来の
 *   GET /mint/mints?inspectionIds=a,b,c (&view=list)
 * を試し、500 等で落ちた場合は
 *   inspections から得た mintId を想定して /mint/mints/{mintId} を並列取得する。
 */
export async function fetchMintListRowsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  // まず view=list を試す → backend 未対応/500なら view なし
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
      out[keys[0]],
    );
    return out;
  } catch (e: any) {
    // ✅ ここが今回の本命（/mint/mints?inspectionIds=... が 500 の時）
    log(
      "fetchMintListRowsByInspectionIdsHTTP fallback to per-mint fetch because:",
      e?.message ?? e,
    );
    return await fetchMintListRowsByMintIdsFallback(ids);
  }
}

/**
 * ✅ 詳細DTO用: inspectionIds (= productionIds) をまとめて渡して、mints(MintDTO) を取得する。
 *
 * backend: GET /mint/mints?inspectionIds=a,b,c  (＋可能なら &view=dto)
 *
 * NOTE:
 * - このルートが 500 の場合もあり得るが、現状の画面要件は list row のみなので
 *   ここは従来通り（必要になったら MintID フォールバックも追加可能）
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
    out[keys[0]],
  );
  return out;
}

/**
 * ✅ 追加: “listMintsByInspectionIDs” という名前で取得したい場合のエイリアス
 * - 画面（service/hook）からはこちらを呼ぶ想定でもOK
 *
 * NOTE:
 * - 呼び出し側が inspections の mintId[] を渡しても動く（fallback があるため）
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

/**
 * 単発: 互換用（既存呼び出しを壊さない）
 * backend: GET /mint/mints/{id}
 *
 * NOTE:
 * - ここでは id を mintId として扱う（inspections の mintId を渡す前提）
 */
export async function fetchMintByInspectionIdHTTP(
  inspectionId: string,
): Promise<MintDTO | null> {
  return await fetchMintByMintIdHTTP(inspectionId);
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
