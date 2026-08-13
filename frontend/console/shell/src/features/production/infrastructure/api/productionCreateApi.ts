// frontend/console/shell/src/features/production/infrastructure/api/productionCreateApi.ts

import type { Brand } from "../../../../shared/types/brand";
import { API_BASE } from "../../../../shared/http/apiBase";
import { fetchJSON } from "../../../../shared/http/fetchJSON";
import { brandRepositoryHTTP } from "../../../brand/infrastructure/http/brandRepositoryHTTP";
import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../productBlueprint/domain/productBlueprintCategory";
import {
  fetchProductBlueprintManagementRows,
  type ProductBlueprintManagementRow,
} from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import type { ProductionQuantityRow } from "../../application/productionQuantityRow";

export type ProductionCreateProductBlueprintResponse = {
  id: string;
  productName: string;
  brandId: string;
  brandName: string;
  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;
  categoryFields?: CategoryFieldValues | null;
  assigneeId?: string;
};

export type ProductionCreateContextResponse = {
  productBlueprintPatch: ProductionCreateProductBlueprintResponse;
  rows: ProductionQuantityRow[];
};

export type {
  Brand,
  ProductBlueprintManagementRow,
  ProductBlueprintCategorySnapshot,
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
// Production Create BFF
// ======================================================================

export async function loadProductionCreateContext(
  productBlueprintId: string,
): Promise<ProductionCreateContextResponse> {
  const normalizedProductBlueprintId = String(productBlueprintId ?? "").trim();

  if (!normalizedProductBlueprintId) {
    throw new Error("loadProductionCreateContext: productBlueprintId が空です");
  }

  const url = `${API_BASE}/productions/create-context?productBlueprintId=${encodeURIComponent(
    normalizedProductBlueprintId,
  )}`;

  return fetchJSON(url, {
    method: "GET",
    auth: "required",
  });
}