// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintDetailApi.ts

/// <reference types="vite/client" />

import { getAuthHeaders } from "../../../../shell/src/auth/application/authService";
import type { SizeRow } from "../../../../model/src/domain/entity/catalog";

// ------------------------------------------------------
// BASE URL（ファイル冒頭に移動して shadowing を防止）
// ------------------------------------------------------
export const API_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)
    ?.replace(/\/+$/g, "") ?? "";

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
  /** モデルバリエーションのバージョン（新規は 1 から） */
  version?: number;
  rgb?: number;
  measurements: NewModelVariationMeasurements;
};

export type ModelNumberRow = {
  size: string;
  color: string;
  code: string;
};

// ✅ ProductBlueprint の modelRefs（displayOrder の唯一のソース）
export type ProductBlueprintModelRef = {
  modelId: string;
  displayOrder: number;
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

  /** ✅ printed を backend 正として受け取る（bool） */
  printed?: boolean | null;

  /** ✅ name 解決済み（backend 正） */
  brandName?: string | null;
  assigneeName?: string | null;
  createdByName?: string | null;

  createdBy?: string | null;
  createdAt?: string | null;
  updatedAt?: string | null;
  deletedAt?: string | null;

  /**
   * ✅ displayOrder に従って並べ替えるために必須
   * backend: toDetailOutput が返す modelRefs をそのまま受ける
   */
  modelRefs?: ProductBlueprintModelRef[];

  /**
   * 互換のため残して良いが、並び順の正は modelRefs 側に寄せる。
   * （modelVariations は別 API で取得して join する設計の方が堅い）
   */
  modelVariations?: Array<{
    id?: string;
    size?: string;
    color?: string;
    modelNumber?: string;
    rgb?: number | null;
    measurements?: Record<string, number | null>;
    productBlueprintId?: string;
    /** 現在のバージョン番号 */
    version?: number;
    createdAt?: string | null;
    updatedAt?: string | null;
  }>;
};

// 更新用パラメータ（application/service が利用する型）
export type UpdateProductBlueprintParams = {
  id: string;

  productName: string;
  brandId: string;
  itemType: string;
  fit: string;
  material: string;
  weight: number;
  qualityAssurance: string[];

  /** ✅ backend DTO に合わせ、repository 側で productIdTag に変換して送る */
  productIdTagType: string | null;

  companyId: string;
  assigneeId: string;

  /** variations / colors は ProductBlueprint 更新 endpoint に送らない（別系で更新） */
  colors: string[];
  colorRgbMap?: Record<string, string>;

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

  const trimmed = String(id ?? "").trim();
  if (!trimmed) {
    throw new Error("getProductBlueprintDetailApi: id が空です");
  }

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(trimmed)}`;

  console.log("[productBlueprintDetailApi] GET detail:", url);

  const json = (await fetchJSON(url, {
    method: "GET",
    headers,
  })) as ProductBlueprintDetailResponse;

  console.log("[productBlueprintDetailApi] detail response:", json);

  return json;
}

/**
 * ✅ UPDATE は repository に集約する方針のため、
 * updateProductBlueprintApi は削除しました。
 * - update は productBlueprintRepositoryHTTP.updateProductBlueprintHTTP を利用してください。
 */
