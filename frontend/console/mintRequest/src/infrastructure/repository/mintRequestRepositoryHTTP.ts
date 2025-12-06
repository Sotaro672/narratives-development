// frontend/console/mintRequest/src/infrastructure/repository/mintRequestRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import type { InspectionBatchDTO } from "../api/mintRequestApi";
import type {
  ProductBlueprintPatchDTO,
  BrandForMintDTO, // ★ 追加
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
  console.log(
    "[mintRequestRepositoryHTTP] fetchInspectionBatchesHTTP url =",
    url,
  );

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  console.log(
    "[mintRequestRepositoryHTTP] fetchInspectionBatchesHTTP status =",
    res.status,
    res.statusText,
  );

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspections (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO[] | null | undefined;
  console.log(
    "[mintRequestRepositoryHTTP] fetchInspectionBatchesHTTP json =",
    json,
  );

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
  console.log(
    "[mintRequestRepositoryHTTP] fetchInspectionByProductionIdHTTP url =",
    url,
  );

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  console.log(
    "[mintRequestRepositoryHTTP] fetchInspectionByProductionIdHTTP status =",
    res.status,
    res.statusText,
  );

  if (res.status === 404) {
    console.log(
      "[mintRequestRepositoryHTTP] fetchInspectionByProductionIdHTTP 404 (not found)",
    );
    return null;
  }

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspection by productionId: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO | null | undefined;
  console.log(
    "[mintRequestRepositoryHTTP] fetchInspectionByProductionIdHTTP json =",
    json,
  );
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

  console.log(
    "[mintRequestRepositoryHTTP] fetchProductBlueprintPatchHTTP url =",
    url,
    "productBlueprintId =",
    productBlueprintId,
  );

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  console.log(
    "[mintRequestRepositoryHTTP] fetchProductBlueprintPatchHTTP status =",
    res.status,
    res.statusText,
  );

  if (res.status === 404) {
    console.log(
      "[mintRequestRepositoryHTTP] fetchProductBlueprintPatchHTTP 404 (not found)",
    );
    return null;
  }

  if (!res.ok) {
    throw new Error(
      `Failed to fetch productBlueprintPatch: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as ProductBlueprintPatchDTO | null | undefined;
  console.log(
    "[mintRequestRepositoryHTTP] fetchProductBlueprintPatchHTTP json =",
    json,
    "brandId =",
    json?.brandId,
    "brandName =",
    json?.brandName,
  );

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
  console.log("[mintRequestRepositoryHTTP] fetchBrandsForMintHTTP url =", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  console.log(
    "[mintRequestRepositoryHTTP] fetchBrandsForMintHTTP status =",
    res.status,
    res.statusText,
  );

  if (!res.ok) {
    throw new Error(
      `Failed to fetch brands (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as BrandPageResultDTO | null | undefined;
  console.log(
    "[mintRequestRepositoryHTTP] fetchBrandsForMintHTTP raw json =",
    json,
  );

  const rawItems: BrandRecordRaw[] = json?.items ?? json?.Items ?? [];
  console.log(
    "[mintRequestRepositoryHTTP] fetchBrandsForMintHTTP raw items =",
    rawItems,
  );

  const mapped: BrandForMintDTO[] = rawItems
    .map((b) => ({
      id: b.id ?? b.ID ?? "",
      name: b.name ?? b.Name ?? "",
    }))
    .filter((b) => b.id && b.name);

  console.log(
    "[mintRequestRepositoryHTTP] fetchBrandsForMintHTTP mapped =",
    mapped,
  );

  return mapped;
}
