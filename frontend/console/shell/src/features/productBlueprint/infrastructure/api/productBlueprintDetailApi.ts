// frontend/console/shell/src/features/productBlueprint/infrastructure/api/productBlueprintDetailApi.ts

import {
  API_BASE,
} from "../../../../shared/http/apiBase";

import {
  fetchJSON,
} from "../../../../shared/http/fetchJSON";

import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../domain/productBlueprintCategory";

export type {
  UpdateProductBlueprintParams,
} from "./productBlueprintUpdateApi";

export type ProductBlueprintModelRef = {
  modelId: string;
  displayOrder: number;
};

export type ProductBlueprintModelVariationResponse = {
  id?: string;
  productBlueprintId?: string;

  kind?:
    | "apparel"
    | "alcohol"
    | string;

  modelNumber?: string;

  size?: string;

  color?:
    | string
    | {
        name?: string;
        rgb?: number | null;
      };

  rgb?: number | null;

  measurements?: Record<
    string,
    number | null
  >;

  volume?: {
    value?: number | null;
    unit?: string | null;
  } | null;

  version?: number;

  createdAt?: string | null;
  updatedAt?: string | null;
};

export type ProductBlueprintDetailResponse = {
  id: string;
  productName: string;

  description?: string | null;

  companyId?: string;

  brandId: string;
  brandName?: string | null;

  productBlueprintCategoryId:
    string;

  productBlueprintCategory:
    ProductBlueprintCategorySnapshot;

  categoryFields?:
    | CategoryFieldValues
    | null;

  productIdTag?: {
    type?: string | null;
  } | null;

  assigneeId?: string;
  assigneeName?: string | null;

  printed?: boolean | null;

  createdBy?: string | null;
  createdByName?: string | null;
  createdAt?: string | null;

  updatedBy?: string | null;
  updatedByName?: string | null;
  updatedAt?: string | null;

  deletedAt?: string | null;

  modelRefs?:
    ProductBlueprintModelRef[];

  modelVariations?:
    ProductBlueprintModelVariationResponse[];
};

export type {
  ProductBlueprintCategorySnapshot,
};

export async function getProductBlueprintDetailApi(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const normalizedId =
    String(
      id ?? "",
    ).trim();

  if (!normalizedId) {
    throw new Error(
      "getProductBlueprintDetailApi: id が空です",
    );
  }

  const url =
    `${API_BASE}/product-blueprints/${encodeURIComponent(
      normalizedId,
    )}`;

  return fetchJSON<ProductBlueprintDetailResponse>(
    url,
    {
      method: "GET",
      auth: "required",
    },
  );
}