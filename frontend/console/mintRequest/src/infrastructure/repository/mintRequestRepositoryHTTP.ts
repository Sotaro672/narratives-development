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

// ===============================
// HTTP Repository (inspections)
// ===============================

/**
 * 現在ログイン中の companyId を起点に、
 * /mint/inspections から inspections の一覧を取得する。
 */
export async function fetchInspectionBatchesHTTP(): Promise<InspectionBatchDTO[]> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/inspections`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspections (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO[] | null | undefined;
  return json ?? [];
}

/**
 * 個別 productionId の InspectionBatch を取得
 * （こちらは従来どおり /products/inspections?productionId=... を使用）
 */
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

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (res.status === 404) {
    return null;
  }

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspection by productionId: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO | null | undefined;
  return json ?? null;
}

// ===============================
// HTTP Repository (productBlueprint Patch)
// ===============================

/**
 * productBlueprintId → ProductBlueprint Patch を取得
 * backend: GET /mint/product_blueprints/{id}/patch
 */
export async function fetchProductBlueprintPatchHTTP(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/product_blueprints/${encodeURIComponent(
    productBlueprintId,
  )}/patch`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
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

/**
 * current companyId に紐づく Brand 一覧を取得する。
 * backend: GET /mint/brands
 *
 * Go 側は branddom.PageResult[branddom.Brand] を返す想定なので、
 * JSON の Items / items から id / name だけを抜き出して BrandForMintDTO[] に変換する。
 */
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
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
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

/**
 * 指定した brandId に紐づく TokenBlueprint 一覧を取得する。
 * backend: GET /mint/token_blueprints?brandId=...
 */
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
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
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

/**
 * inspectionIds (= productionIds) をまとめて渡して、mints(list row) を取得する。
 * backend: GET /mint/mints?inspectionIds=a,b,c
 *
 * 戻り値は "inspectionId -> MintListRowDTO" の map を期待（画面側での突合を簡単にするため）。
 *
 * NOTE:
 * - 一覧画面の結合用に必要なのは tokenName / createdByName / mintedAt のみ
 * - それ以外は detail API が担う
 */
export async function fetchMintsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const idToken = await getIdTokenOrThrow();

  // query: inspectionIds=... (CSV)
  const url = `${API_BASE}/mint/mints?inspectionIds=${encodeURIComponent(
    ids.join(","),
  )}`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (res.status === 404) {
    return {};
  }

  if (!res.ok) {
    throw new Error(`Failed to fetch mints: ${res.status} ${res.statusText}`);
  }

  // 期待: { [inspectionId]: MintListRowDTO }
  const json = (await res.json()) as
    | Record<string, MintListRowDTO>
    | null
    | undefined;

  return json ?? {};
}

/**
 * 単発: inspectionId (= productionId) で 1 件取得（バックエンドが用意されている場合）
 * backend: GET /mint/mints/{inspectionId}
 *
 * NOTE:
 * - 既存実装が詳細用の MintDTO を返す前提のまま残す
 * - 一覧は fetchMintsByInspectionIdsHTTP（MintListRowDTO）を使う
 */
export async function fetchMintByInspectionIdHTTP(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) {
    throw new Error("inspectionId が空です");
  }

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/mints/${encodeURIComponent(iid)}`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (res.status === 404) {
    return null;
  }

  if (!res.ok) {
    throw new Error(
      `Failed to fetch mint by inspectionId: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as MintDTO | null | undefined;
  return json ?? null;
}

// ===============================
// HTTP Repository (mint request)
// ===============================

/**
 * ミント申請リクエストを送信する。
 * backend: POST /mint/inspections/{productionId}/request
 *
 * Body:
 * {
 *   "tokenBlueprintId": "...",
 *   "scheduledBurnDate": "YYYY-MM-DD" // 任意
 * }
 */
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

  // HTML の date input から渡される "YYYY-MM-DD" をそのまま送る
  if (scheduledBurnDate && scheduledBurnDate.trim()) {
    payload.scheduledBurnDate = scheduledBurnDate.trim();
  }

  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (res.status === 404) {
    return null;
  }

  if (!res.ok) {
    throw new Error(
      `Failed to post mint request: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO | null | undefined;
  return json ?? null;
}
