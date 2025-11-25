// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintDetailApi.ts

/// <reference types="vite/client" />

import { getAuthHeaders } from "../../../../shell/src/auth/application/authService";
import type { SizeRow } from "../../../../model/src/domain/entity/catalog";

// ------------------------------------------------------
// 型定義
// ------------------------------------------------------

// 採寸キー → 数値 or null
export type NewModelVariationMeasurements = Record<string, number | null>;

// モデルバリエーション 1 行分
export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  modelNumber: string;
  createdBy: string;
  rgb?: number;
  measurements: NewModelVariationMeasurements;
};

export type ModelNumberRow = {
  size: string;
  color: string;
  code: string;
};

// ProductBlueprint 詳細レスポンス
export type ProductBlueprintDetailResponse = {
  id: string;
  productName: string;
  brandId: string;
  itemType: string;
  fit: string;
  material: string;
  weight: number;
  qualityAssurance: string[];

  productIdTag?: {
    type?: string | null;
  } | null;

  companyId?: string;
  assigneeId?: string;

  createdBy?: string | null;
  createdAt?: string | null;
  updatedAt?: string | null;
  deletedAt?: string | null;

  modelVariations?: Array<{
    id?: string;
    size?: string;
    color?: string;
    modelNumber?: string;
    rgb?: number | null;
    measurements?: Record<string, number | null>;
    productBlueprintId?: string;
    createdAt?: string | null;
    updatedAt?: string | null;
  }>;
};

// 更新用パラメータ
export type UpdateProductBlueprintParams = {
  id: string;

  productName: string;
  brandId: string;
  itemType: string; // ← string のままで OK（service 側で ItemType にキャストする）
  fit: string;
  material: string;
  weight: number;
  qualityAssurance: string[];

  productIdTagType: string | null;

  companyId: string;
  assigneeId: string;

  colors: string[];
  colorRgbMap?: Record<string, string>;

  // ← ここから追加分（service 側だけで使うフィールド）
  sizes?: SizeRow[];
  modelNumbers?: ModelNumberRow[];
  updatedBy?: string | null;
};

// ------------------------------------------------------
// 共通 fetch ヘルパー
// ------------------------------------------------------
async function fetchJSON(input: RequestInfo, init?: RequestInit) {
  const res = await fetch(input, init);
  const ct = res.headers.get("content-type") ?? "";

  if (!ct.includes("application/json")) {
    const text = await res.text().catch(() => "");
    throw new Error(`Unexpected content-type: ${ct}\n${text.slice(0, 200)}`);
  }

  if (!res.ok) {
    const text = await res.text().catch(() => `HTTP ${res.status}`);
    throw new Error(text);
  }

  return res.json();
}

// ------------------------------------------------------
// GET /product-blueprints/{id}  詳細取得
// ------------------------------------------------------
export async function getProductBlueprintDetailApi(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const headers = await getAuthHeaders();

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(id)}`;

  console.log("[productBlueprintDetailApi] GET detail:", url);

  const json = (await fetchJSON(url, {
    method: "GET",
    headers,
  })) as ProductBlueprintDetailResponse;

  console.log("[productBlueprintDetailApi] detail response:", json);

  return json;
}

// ------------------------------------------------------
// PATCH /product-blueprints/{id}  更新
// ------------------------------------------------------
const API_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

export async function updateProductBlueprintApi(
  params: UpdateProductBlueprintParams,
  variations: NewModelVariationPayload[],
): Promise<ProductBlueprintDetailResponse> {
  const headers = {
    ...(await getAuthHeaders()),
    "Content-Type": "application/json",
  };

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(params.id)}`;

  const payload = {
    productName: params.productName,
    brandId: params.brandId,
    itemType: params.itemType,
    fit: params.fit,
    material: params.material,
    weight: params.weight,
    qualityAssurance: params.qualityAssurance,
    productIdTagType: params.productIdTagType,
    companyId: params.companyId,
    assigneeId: params.assigneeId,
    colors: params.colors,
    colorRgbMap: params.colorRgbMap ?? {},
    variations,
  };

  console.log("[productBlueprintDetailApi] PATCH payload:", {
    url,
    payload,
  });

  const json = (await fetchJSON(url, {
    method: "PATCH",
    headers,
    body: JSON.stringify(payload),
  })) as ProductBlueprintDetailResponse;

  console.log("[productBlueprintDetailApi] update response:", json);

  return json;
}
