// frontend/console/mintRequest/src/infrastructure/repository/mintRequestRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import type { InspectionBatchDTO } from "../api/mintRequestApi";
import type {
  ProductBlueprintPatchDTO,
  BrandForMintDTO,
  TokenBlueprintForMintDTO, // ★ 追加
} from "../../application/mintRequestService";

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
  items?: BrandRecordRaw[]; // 将来 json タグを付けた場合
  Items?: BrandRecordRaw[]; // 現状の Go デフォルト (先頭大文字)
  // 他に total / page / perPage 等があっても無視する
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
      id: b.id ?? b.ID ?? "",
      name: b.name ?? b.Name ?? "",
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
 *
 * Go 側は tbdom.PageResult を返す想定なので、
 * JSON の Items / items から id / name / symbol / iconUrl を抜き出して
 * TokenBlueprintForMintDTO[] に変換する。
 */
type TokenBlueprintRecordRaw = {
  id?: string;
  name?: string;
  symbol?: string;
  iconUrl?: string;

  // 大文字始まりのフィールドにも一応対応
  ID?: string;
  Name?: string;
  Symbol?: string;
  IconUrl?: string;
};

type TokenBlueprintPageResultDTO = {
  items?: TokenBlueprintRecordRaw[];
  Items?: TokenBlueprintRecordRaw[];
  // total / page / perPage などは無視
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

  let rawItems: TokenBlueprintRecordRaw[] = [];

  if (Array.isArray(json)) {
    // handler が単純に []TokenBlueprint を返すケース
    rawItems = json;
  } else {
    // PageResult 経由で返ってくるケース
    rawItems = json?.items ?? json?.Items ?? [];
  }

  const mapped: TokenBlueprintForMintDTO[] = rawItems
    .map((tb) => ({
      id: tb.id ?? tb.ID ?? "",
      name: tb.name ?? tb.Name ?? "",
      symbol: tb.symbol ?? tb.Symbol ?? "",
      iconUrl: tb.iconUrl ?? tb.IconUrl ?? undefined,
    }))
    .filter((tb) => tb.id && tb.name && tb.symbol);

  return mapped;
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
 *   "tokenBlueprintId": "..."
 * }
 */
export async function postMintRequestHTTP(
  productionId: string,
  tokenBlueprintId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) {
    throw new Error("productionId が空です");
  }

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(
    trimmed,
  )}/request`;

  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      tokenBlueprintId,
    }),
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
