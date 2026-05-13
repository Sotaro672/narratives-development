// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintDetailApi.ts

import { getAuthHeaders } from "../../../../shell/src/auth/application/authService";
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { fetchJSON } from "../../../../shell/src/shared/http/fetchJSON";

import type {
  ApparelModelNumberRow,
  ApparelSizeInput,
} from "../../domain/entity/apparel";

import type {
  CategoryFieldValues,
  ProductBlueprintCategoryKind,
  ProductBlueprintCategorySnapshot,
} from "../../domain/entity/productBlueprintCategory";

// ------------------------------------------------------
// apparel model variation 共通型
// ------------------------------------------------------

export type ApparelModelVariationResponse = {
  id?: string;
  size?: string;
  color?: string;
  modelNumber?: string;
  rgb?: number | null;
  measurements?: Record<string, number | null>;
  productBlueprintId?: string;
  version?: number;
  createdAt?: string | null;
  updatedAt?: string | null;
};

export type ProductBlueprintModelRef = {
  modelId: string;
  displayOrder: number;
};

// ------------------------------------------------------
// ProductBlueprint 詳細レスポンス
// ------------------------------------------------------

export type ProductBlueprintDetailResponse = {
  id: string;
  productName: string;

  brandId: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;

  fit: string;
  material: string;
  weight: number;
  qualityAssurance: string[];

  productIdTag?: {
    type?: string | null;
  } | null;

  companyId?: string;
  assigneeId?: string;

  printed?: boolean | null;

  brandName?: string | null;
  assigneeName?: string | null;
  createdByName?: string | null;

  createdBy?: string | null;
  createdAt?: string | null;
  updatedAt?: string | null;
  deletedAt?: string | null;

  modelRefs?: ProductBlueprintModelRef[];

  modelVariations?: ApparelModelVariationResponse[];

  categoryFields?: CategoryFieldValues | null;
};

// ------------------------------------------------------
// 更新用パラメータ
// ------------------------------------------------------

export type UpdateProductBlueprintParams = {
  id: string;

  productName: string;
  brandId: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;

  fit: string;
  material: string;
  weight: number;
  qualityAssurance: string[];

  productIdTagType: string | null;

  companyId: string;
  assigneeId: string;

  colors: string[];
  colorRgbMap?: Record<string, string>;

  sizes?: ApparelSizeInput[];
  modelNumbers?: ApparelModelNumberRow[];
  updatedBy?: string | null;

  categoryFields?: CategoryFieldValues | null;
};

export type { ProductBlueprintCategoryKind, ProductBlueprintCategorySnapshot };

// ------------------------------------------------------
// GET /product-blueprints/{id} 詳細取得
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

  const json = await fetchJSON<ProductBlueprintDetailResponse>(url, {
    method: "GET",
    headers,
  });

  return json;
}