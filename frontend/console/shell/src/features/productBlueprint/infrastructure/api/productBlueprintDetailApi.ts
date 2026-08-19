// frontend/console/shell/src/features/productBlueprint/infrastructure/api/productBlueprintDetailApi.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { fetchJSON } from "../../../../shared/http/fetchJSON";
import type {
  CategoryFieldValues,
  ProductBlueprintCategoryPath,
} from "../../domain/productBlueprintCategory";

export type { UpdateProductBlueprintParams } from "./productBlueprintUpdateApi";

export type ProductBlueprintDetailSizeResponse = {
  id: string;
  sizeLabel: string;
  length?: number;
  width?: number;
  chest?: number;
  shoulder?: number;
  sleeveLength?: number;
  waist?: number;
  hip?: number;
  rise?: number;
  inseam?: number;
  thigh?: number;
  hemWidth?: number;
};

export type ProductBlueprintDetailShippingPackageResponse = {
  weightGrams: number;
  widthMm: number;
  lengthMm: number;
  heightMm: number;
};

export type ProductBlueprintDetailApparelModelNumberResponse = {
  size: string;
  color: string;
  code: string;
  shippingPackage: ProductBlueprintDetailShippingPackageResponse;
};

export type ProductBlueprintDetailVolumeResponse = {
  value: number;
  unit: string;
};

export type ProductBlueprintDetailVolumeRowResponse = {
  id: string;
  volumeValue: number;
  volumeUnit: string;
};

export type ProductBlueprintDetailAlcoholModelNumberResponse = {
  kind: "alcohol";
  volume: ProductBlueprintDetailVolumeResponse;
  code: string;
  shippingPackage: ProductBlueprintDetailShippingPackageResponse;
};

export type ProductBlueprintDetailModelStateResponse = {
  colors: string[];
  sizes: ProductBlueprintDetailSizeResponse[];
  modelNumbers: ProductBlueprintDetailApparelModelNumberResponse[];
  colorRgbMap: Record<string, string>;
  volumes: ProductBlueprintDetailVolumeRowResponse[];
  alcoholModelNumbers: ProductBlueprintDetailAlcoholModelNumberResponse[];
};

export type ProductBlueprintDetailResponse = {
  id: string;
  productName: string;
  description: string;
  companyId: string;
  brandId: string;
  brandName: string;
  productBlueprintCategoryPath: ProductBlueprintCategoryPath;
  categoryFields?: CategoryFieldValues;
  productIdTag?: { type: string };
  assigneeId: string;
  assigneeName: string;
  printed: boolean;
  createdBy: string;
  createdByName: string;
  createdAt: string;
  updatedBy: string;
  updatedByName: string;
  updatedAt: string;
  modelState: ProductBlueprintDetailModelStateResponse;
};

export type { ProductBlueprintCategoryPath };

export async function getProductBlueprintDetailApi(id: string): Promise<ProductBlueprintDetailResponse> {
  if (!id) {
    throw new Error("getProductBlueprintDetailApi: id が空です");
  }

  return fetchJSON<ProductBlueprintDetailResponse>(
    `${API_BASE}/product-blueprints/${encodeURIComponent(id)}`,
    { method: "GET", auth: "required" },
  );
}