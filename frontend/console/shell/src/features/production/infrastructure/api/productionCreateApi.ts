// frontend/console/shell/src/features/production/infrastructure/api/productionCreateApi.ts

import type { Brand } from "../../../../shared/types/brand";
import type { ProductBlueprintManagementRow } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import type { ProductBlueprintDetailResponse } from "../../../productBlueprint/infrastructure/api/productBlueprintDetailApi";
import type { ModelVariationResponse } from "../../../productBlueprint/application/productBlueprintDetailService";

import { brandRepositoryHTTP } from "../../../brand/infrastructure/http/brandRepositoryHTTP";
import { fetchProductBlueprintManagementRows } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
} from "../../../productBlueprint/application/productBlueprintDetailService";

export type {
  Brand,
  ProductBlueprintManagementRow,
  ProductBlueprintDetailResponse,
  ModelVariationResponse,
};

// ======================================================================
// ブランドAPI
// ======================================================================
export async function loadBrands(): Promise<Brand[]> {
  try {
    const result = await brandRepositoryHTTP.list({
      page: 1,
      perPage: 200,
    });

    return result.items.filter((brand) => brand.isActive);
  } catch {
    return [];
  }
}

// ======================================================================
// 商品設計一覧API
// ======================================================================
export async function loadProductBlueprints(): Promise<ProductBlueprintManagementRow[]> {
  try {
    return await fetchProductBlueprintManagementRows();
  } catch {
    return [];
  }
}

// ======================================================================
// 商品設計詳細 + ModelVariations API
// ======================================================================
export async function loadDetailAndModels(
  productBlueprintId: string,
): Promise<{
  detail: ProductBlueprintDetailResponse;
  models: ModelVariationResponse[];
}> {
  const [detail, models] = await Promise.all([
    getProductBlueprintDetail(productBlueprintId),
    listModelVariationsByProductBlueprintId(productBlueprintId),
  ]);

  return {
    detail,
    models,
  };
}